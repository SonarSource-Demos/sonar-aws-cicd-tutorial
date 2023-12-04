package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"CDK/pkg/mainconfig"

	"gopkg.in/yaml.v2"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/briandowns/spinner"
	"github.com/golang/glog"

	"github.com/go-git/go-git/v5"
	gitconfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/retry"
)

const configMapYAML1 = `
    - rolearn: %s
      username: admin
      groups:
        - system:masters
`

type BuildSpec struct {
	Version string `yaml:"version"`
	Env     struct {
		SecretsManager struct {
			SonarToken   string `yaml:"SONAR_TOKEN"`
			SonarHostURL string `yaml:"SONAR_HOST_URL"`
		} `yaml:"secrets-manager"`
		Variables struct {
			SourceBranch       string `yaml:"SourceBranch"`
			DestinationBranch  string `yaml:"DestinationBranch"`
			ImageRepoName      string `yaml:"IMAGE_REPO_NAME"`
			ImageTag           string `yaml:"IMAGE_TAG"`
			EKSClusterName     string `yaml:"EKS_CLUSTER_NAME"`
			EKSNSApp           string `yaml:"EKS_NS_APP"`
			EKSCodeBuildAppSvc string `yaml:"EKS_CODEBUILD_APP_SVC"`
			EKSDeployApp       string `yaml:"EKS_DEPLOY_APP"`
			EKSRole            string `yaml:"EKS_ROLE"`
			SonarProject       string `yaml:"SONAR_PROJECT"`
			PRKey              string `yaml:"PRKey"`
		} `yaml:"variables"`
		Shell string `yaml:"shell"`
	} `yaml:"env"`
	Phases struct {
		Install struct {
			RuntimeVersions struct {
				Java string `yaml:"java"`
			} `yaml:"runtime-versions"`
		} `yaml:"install"`
		PreBuild struct {
			Commands []string `yaml:"commands"`
		} `yaml:"pre_build"`
		Build struct {
			Commands []string `yaml:"commands"`
		} `yaml:"build"`
		PostBuild struct {
			Commands []string `yaml:"commands"`
		} `yaml:"post_build"`
	} `yaml:"phases"`
	Artifacts struct {
		Files        []string `yaml:"files"`
		DiscardPaths bool     `yaml:"discard-paths"`
	} `yaml:"artifacts"`
}

type Configuration struct {
	Reponame         string
	Desc             string
	GitRepo          string
	Recr             string
	ImgTag           string
	BuildPr          string
	PiplineN         string
	ClusterName      string
	EksAdminRole     string
	SecondBramchName string
}

func readJSONConfig(filename string, config interface{}) {
	fconfig, err := os.ReadFile(filename)
	if err != nil {
		panic(fmt.Sprintf("❌ Problem with the configuration file: %s", filename))
		os.Exit(1)
	}
	if err := json.Unmarshal(fconfig, config); err != nil {
		fmt.Println("❌ Error unmarshaling JSON:", err)
		os.Exit(1)
	}
}

func GetConfig(configcrd mainconfig.ConfAuth, configjs Configuration) (mainconfig.ConfAuth, Configuration) {

	readJSONConfig("config.json", &configjs)
	readJSONConfig("../config_crd.json", &configcrd)

	return configcrd, configjs
}

func updateAwsAuthConfigMap(clientset *kubernetes.Clientset, rolearn string) {
	configMapName := "aws-auth"
	namespace := "kube-system"
	//namespace := "test"

	configMapYAMLWithARN := fmt.Sprintf(configMapYAML1, rolearn)

	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		configMap, getErr := clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), configMapName, metav1.GetOptions{})
		if getErr != nil {
			return getErr
		}

		// Get the current value of mapRoles
		currentValue, _ := configMap.Data["mapRoles"]

		configMap.Data["mapRoles"] = currentValue + "\n" + configMapYAMLWithARN

		_, updateErr := clientset.CoreV1().ConfigMaps(namespace).Update(context.TODO(), configMap, metav1.UpdateOptions{})
		return updateErr
	})

	if err != nil {
		panic(err.Error())
	}
}

func CheckIfError(err error) {
	if err == nil {
		return
	}

	fmt.Printf("\x1b[31;1m%s\x1b[0m\n", fmt.Sprintf(" ❌ Error: %s", err))
	os.Exit(1)
}

func waitForCodeCommitCreation(repo string) {
	spin := spinner.New(spinner.CharSets[37], 100*time.Millisecond, spinner.WithWriter(os.Stderr))
	spin.Suffix = " Waiting for CodeCommit repository creation..."
	spin.Start()

	for {
		// Use the AWS CLI or SDK to check for the existence of the CodeCommit repository
		cmd := exec.Command("aws", "codecommit", "get-repository", "--repository-name", repo)
		err := cmd.Run()

		if err == nil {
			// Repository exists, break out of the loop
			spin.Stop()
			fmt.Printf("✅ CodeCommit repository created successful.\n")

			break
		}

		fmt.Println("CodeCommit repository not yet created. Retrying in 10 seconds...")
		time.Sleep(10 * time.Second)
	}
}

func getCurrentClusterName(config *rest.Config, kubeconfigPath string) (string, error) {
	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath},
		&clientcmd.ConfigOverrides{ClusterInfo: clientcmdapi.Cluster{Server: config.Host}},
	)

	rawConfig, err := clientConfig.RawConfig()
	if err != nil {
		return "", fmt.Errorf("failed to load kubeconfig: %v", err)
	}

	// Get the current context
	currentContext := rawConfig.CurrentContext
	if currentContext == "" {
		return "", fmt.Errorf("current context not set in kubeconfig")
	}

	// Get the cluster name for the current context
	clusterARN := rawConfig.Contexts[currentContext].Cluster
	if clusterARN == "" {
		return "", fmt.Errorf("cluster name not found for current context")
	}

	clusterName := getLastSegmentAfterSlash(clusterARN)

	return clusterName, nil
}

func getLastSegmentAfterSlash(s string) string {
	parts := strings.Split(s, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

func main() {

	var configcrd mainconfig.ConfAuth
	var config1 Configuration
	var AppConfig1, AppConfig = GetConfig(configcrd, config1)
	var roleArn string
	RepoNameCd := AppConfig.Reponame + "-" + AppConfig1.Index
	ERCReposName := AppConfig.Recr + "-" + AppConfig1.Index
	secretName := AppConfig1.AWSsecret + AppConfig1.Index
	BuilRole := "BuildAdminRole" + AppConfig1.Index
	BuildSecretToken := secretName + ":SONAR_TOKEN"
	BuildSecretURL := secretName + ":SONAR_HOST_URL"
	BranchToMerge := "main"
	SecondBramchName := AppConfig.SecondBramchName
	BuildFile := "buildspec.yml"

	clusterName := AppConfig.ClusterName + AppConfig1.Index
	AdmRole := clusterName + AppConfig.EksAdminRole
	//ClusterARNRole := "arn:aws:iam::" + AppConfig1.Account + ":role/" + AdmRole
	buildAdminRoleARN := "arn:aws:iam::" + AppConfig1.Account + ":role/" + BuilRole

	os.Setenv("AWS_SDK_LOAD_CONFIG", "true")
	os.Setenv("AWS_PROFILE", AppConfig1.SSOProfile)
	codeCommitRepoURL := "codecommit://" + AppConfig1.SSOProfile + "@" + RepoNameCd
	filePath := RepoNameCd + "/" + BuildFile

	// wait CodeCommit repo created
	waitForCodeCommitCreation(RepoNameCd)

	stackName := "DevopsStack" + AppConfig1.Index
	os.Setenv("AWS_SDK_LOAD_CONFIG", "true")
	// Create a new AWS session
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(AppConfig1.Region),
	}))

	// Create a IAM client
	svc := iam.New(sess)

	// Get Existing EKS Admin Role
	getRoleInput := &iam.GetRoleInput{
		RoleName: &AdmRole,
	}
	getRoleOutput, err := svc.GetRole(getRoleInput)
	if err != nil {
		fmt.Println("❌ Error getting EKS Admin Role:", err)
		return
	}

	// URL decode the existing trust policy
	existingPolicy, err := url.QueryUnescape(aws.StringValue(getRoleOutput.Role.AssumeRolePolicyDocument))
	if err != nil {
		fmt.Println("❌ Error URL decoding existing policy for EKS Admin Role:", err)
		return
	}

	// Update the existing trust policy by appending the new statement
	existingPolicyMap := make(map[string]interface{})
	err = json.Unmarshal([]byte(existingPolicy), &existingPolicyMap)
	if err != nil {
		fmt.Println("❌ Error unmarshalling existing policy for EKS Admin Role :", err)
		return
	}

	newStatement := map[string]interface{}{
		"Effect": "Allow",
		"Principal": map[string]interface{}{
			"AWS": buildAdminRoleARN,
		},
		"Action": "sts:AssumeRole",
	}

	existingPolicyMap["Statement"] = append(existingPolicyMap["Statement"].([]interface{}), newStatement)

	updatedPolicyJSON, err := json.Marshal(existingPolicyMap)
	if err != nil {
		fmt.Println("❌ Error marshalling updated policy for EKS Admin Role:", err)
		return
	}

	// Update the role's trust policy
	updateAssumeRolePolicyInput := &iam.UpdateAssumeRolePolicyInput{
		RoleName:       &AdmRole,
		PolicyDocument: aws.String(string(updatedPolicyJSON)),
	}

	_, err = svc.UpdateAssumeRolePolicy(updateAssumeRolePolicyInput)
	if err != nil {
		fmt.Println("❌ Error updating EKS Admin Role trust policy:", err)
		return
	}

	fmt.Println("✅ Successfully updated EKS Admin Role.")

	// Create a CloudFormation client
	cfClient := cloudformation.New(sess)

	// Load Kubeconfig
	kubeconfigPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	config, err := rest.InClusterConfig()
	if err != nil {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		glog.Fatalf("❌ Failed to create a ClientSet: %v. Exiting.", err)
	}

	// Get the current context's cluster name
	EKSClusterName, err := getCurrentClusterName(config, kubeconfigPath)
	if err != nil {
		glog.Fatalf("❌ Failed to get cluster name: %v", err)
		os.Exit(1)
	}

	spin1 := spinner.New(spinner.CharSets[37], 100*time.Millisecond, spinner.WithWriter(os.Stderr))
	spin1.Suffix = " Update ConfigMap EKS ..."
	spin1.Start()

	// Obtain a reference to the existing IAM role
	describeStackOutput, err := cfClient.DescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: aws.String(stackName),
	})
	if err != nil {
		fmt.Println("Error describing stack:", err)
		os.Exit(1)
	}

	// Extract outputs from the stack description
	outputs := describeStackOutput.Stacks[0].Outputs

	// Print the outputs
	//	fmt.Println("Stack Outputs:")
	for _, output := range outputs {
		//fmt.Printf("%s: %s\n", *output.OutputKey, *output.OutputValue)
		roleArn = *output.OutputValue
	}

	updateAwsAuthConfigMap(clientset, roleArn)
	spin1.Stop()
	fmt.Println("✅ Successfully updated aws-auth ConfigMap.")

	spin1.Suffix = " Clone GitHub App Java Demo ..."
	spin1.Start()

	// Clone GitHub App Java Demo
	repo1, err1 := git.PlainClone(RepoNameCd, false, &git.CloneOptions{
		URL: AppConfig.GitRepo,
	})
	CheckIfError(err1)

	// Fetch all references (branches) from the remote repository
	err = repo1.Fetch(&git.FetchOptions{
		RemoteName: "origin",
		RefSpecs: []gitconfig.RefSpec{
			gitconfig.RefSpec("+refs/heads/*:refs/heads/*"),
		},
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		CheckIfError(err)
	}
	spin1.Stop()
	fmt.Printf("✅ Clone GitHub App Java Demo is successful.\n")

	/*--------------------- Open Repository --------------------------------*/

	// Open the repository
	repo, err := git.PlainOpen(RepoNameCd)
	if err != nil {
		fmt.Printf("❌ Failed to open repository: %v\n", err)
		os.Exit(1)
	}

	// Get the worktree
	worktree, err := repo.Worktree()
	if err != nil {
		fmt.Printf("❌ Failed to get repo worktree: %v\n", err)
		os.Exit(1)
	}

	/*---------------------- Modify buildspec.yaml ---------------------------*/

	content, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("Error reading file: %v", err)
	}

	// Parse YAML content into a struct
	var buildSpec BuildSpec
	err = yaml.Unmarshal(content, &buildSpec)
	if err != nil {
		log.Fatalf("Error unmarshalling YAML: %v", err)
	}

	// Modify the desired variables

	buildSpec.Env.SecretsManager.SonarToken = BuildSecretToken
	buildSpec.Env.SecretsManager.SonarHostURL = BuildSecretURL
	buildSpec.Env.Variables.ImageRepoName = ERCReposName
	buildSpec.Env.Variables.EKSClusterName = EKSClusterName
	buildSpec.Env.Variables.EKSRole = AdmRole

	// Convert the struct back to YAML
	modifiedYAML, err := yaml.Marshal(&buildSpec)
	if err != nil {
		log.Fatalf("Error marshalling YAML: %v", err)
	}

	// Write the modified YAML back to the file
	err = os.WriteFile(filePath, modifiedYAML, os.ModePerm)
	if err != nil {
		log.Fatalf("Error writing file: %v", err)
	}

	/*---------------------- END Modify buildspec.yaml ---------------------------*/

	/*---------------------- COMMIT change in main branch ------------------------*/

	// Add the changes to the worktree
	_, err = worktree.Add(BuildFile)
	if err != nil {
		fmt.Printf("❌ Failed to add changes to worktree: %v\n", err)
		os.Exit(1)
	}
	status, err := worktree.Status()
	fmt.Printf("✅ Commit : %s", status)

	// Commit the changes
	_, err = worktree.Commit("Update buildspec.yml", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "EC",
			Email: "ec@loclahost.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		fmt.Printf("❌ Failed to commit changes: %v\n", err)
		os.Exit(1)
	}

	cmd2 := exec.Command("git", "checkout", SecondBramchName)
	cmd2.Dir = RepoNameCd
	err1 = cmd2.Run()
	if err != nil {
		spin1.Stop()
		fmt.Println("\n ❌ Error checkout second branch", err)
		os.Exit(1)
	}

	cmd3 := exec.Command("git", "restore", "--source", BranchToMerge, BuildFile)
	cmd3.Dir = RepoNameCd
	err1 = cmd3.Run()
	if err != nil {
		spin1.Stop()
		fmt.Println("\n ❌ Error checkout buildspec.yaml Second Brench:", err)
		os.Exit(1)
	}

	cmd4 := exec.Command("git", "commit", "-a", "-m", "update buildspec.yml")
	cmd4.Dir = RepoNameCd
	err1 = cmd4.Run()
	if err != nil {
		spin1.Stop()
		fmt.Println("\n ❌ Error checkout buildspec.yaml Second Brench:", err)
		os.Exit(1)
	}
	fmt.Printf("✅ Modify buildspec.yaml is successful.\n")

	/*---------------------- Push Repository in CodeCommit Repository ------------------------*/

	spin1.Suffix = "Push Repository in CodeCommit Repository ..."
	spin1.Start()
	// Push Repo in CodeCommit Repo
	cmd := exec.Command("git", "push", "--all", codeCommitRepoURL)
	cmd.Dir = RepoNameCd

	err1 = cmd.Run()
	if err != nil {
		spin1.Stop()
		fmt.Println("\n ❌ Error Push Repository in CodeCommit:", err)
		os.Exit(1)
	}

	spin1.Stop()
	fmt.Printf("✅ Push Repository in CodeCommit Repository is successful.\n")

	// remove a Local directory for repos
	err = os.RemoveAll(RepoNameCd)
	if err != nil {
		fmt.Println("\n ❌ Error Remove A local Repository :", err)
		os.Exit(1)
	}

}
