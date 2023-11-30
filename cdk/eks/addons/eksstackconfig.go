package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awseks"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"

	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"

	//"k8s.io/apimachinery/pkg/util/yaml"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
)

type EksstackconfigStackProps struct {
	awscdk.StackProps
}

type ConfAuth struct {
	Region     string
	Account    string
	SSOProfile string
	Index      string
	AWSsecret  string
}

type Configuration struct {
	ClusterName  string
	VPCid        string
	K8sVersion   string
	Workernode   float64
	EksAdminRole string
	EBSRole      string
	Instance     string
	InstanceSize string
	AddonVersion string
	ScName       string
	ScNamef      string
}

type ClusterProps struct {
	stack       awscdk.Stack
	clusterName string
	region      string
}

type EksClusterWithOIDC struct {
	OidcIssuer string
}

func applyResourcesFromYAML(yamlContent []byte, clientset *kubernetes.Clientset, dd *dynamic.DynamicClient) error {
	decoder := yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(yamlContent), 100)

	for {
		var rawObj runtime.RawExtension
		if err := decoder.Decode(&rawObj); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		obj, gvk, err := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme).Decode(rawObj.Raw, nil, nil)
		if err != nil {
			return err
		}

		unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			return err
		}

		unstructuredObj := &unstructured.Unstructured{Object: unstructuredMap}

		gr, err := restmapper.GetAPIGroupResources(clientset.Discovery())
		if err != nil {
			return err
		}

		mapper := restmapper.NewDiscoveryRESTMapper(gr)
		mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return err
		}

		var dri dynamic.ResourceInterface
		if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
			if unstructuredObj.GetNamespace() == "" {
				unstructuredObj.SetNamespace("default")
			}
			dri = dd.Resource(mapping.Resource).Namespace(unstructuredObj.GetNamespace())
		} else {
			dri = dd.Resource(mapping.Resource)
		}

		_, err = dri.Create(context.Background(), unstructuredObj, metav1.CreateOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}

func EksClusterInfo(scope constructs.Construct, id *string, props *ClusterProps) *EksClusterWithOIDC {

	sess := session.Must(session.NewSession())
	svc := eks.New(sess, aws.NewConfig().WithRegion(props.region))

	for {
		input := &eks.DescribeClusterInput{
			Name: aws.String(props.clusterName),
		}
		result, err := svc.DescribeCluster(input)
		if err != nil {
			fmt.Println("❌ Error describing EKS cluster:", err)
			return nil
		}

		status := aws.StringValue(result.Cluster.Status)

		if status == "ACTIVE" {
			fmt.Println("EKS Cluster is now active.")
			oidcIssuer := aws.StringValue(result.Cluster.Identity.Oidc.Issuer)
			return &EksClusterWithOIDC{
				OidcIssuer: oidcIssuer,
			}
		} else {
			fmt.Println("Cluster status:", status)
		}

		time.Sleep(30 * time.Second)
	}
}

func GetConfig(configcrd ConfAuth, configjs Configuration) (ConfAuth, Configuration) {

	fconfig, err := os.ReadFile("../config.json")
	if err != nil {
		panic("❌ Problem with the configuration file : config.json")
		os.Exit(1)
	}
	if err := json.Unmarshal(fconfig, &configjs); err != nil {
		fmt.Println("❌ Error unmarshaling JSON:", err)
		os.Exit(1)
	}

	fconfig2, err := os.ReadFile("../../config_crd.json")
	if err != nil {
		panic("❌ Problem with the configuration file : config_crd.json")
		os.Exit(1)
	}
	if err := json.Unmarshal(fconfig2, &configcrd); err != nil {
		fmt.Println("❌ Error unmarshaling JSON:", err)
		os.Exit(1)
	}
	return configcrd, configjs
}

func NewEksstackconfigStack(scope constructs.Construct, id string, props *EksstackconfigStackProps, AppConfig Configuration, AppConfig1 ConfAuth, destroy string) awscdk.Stack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)

	// Set Variables
	var clusterName = AppConfig.ClusterName + AppConfig1.Index
	var EbsRole = clusterName + AppConfig.EBSRole

	var policyArn = "arn:aws:iam::aws:policy/service-role/AmazonEBSCSIDriverPolicy"

	eksClusterProps := ClusterProps{
		clusterName: clusterName,
		region:      AppConfig1.Region,
	}

	InfosEks := EksClusterInfo(stack, jsii.String("EKSInfo"), &eksClusterProps)

	oidcIssuer := InfosEks.OidcIssuer
	parts := strings.Split(oidcIssuer, "/")
	// Get the last part (element) from the slice
	OpenID := parts[len(parts)-1]

	/*------------------------------ Connect K8s ---------------------------------------------*/
	// Load Kubeconfig
	kubeconfigPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	config, err := rest.InClusterConfig()
	if err != nil {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	}

	// create kubernetes client
	dd, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		glog.Fatalf("❌ Failed to create a ClientSet: %v. Exiting.", err)
	}

	/*---------------------------End Connect K8s ---------------------------------------------*/

	/*--------------------------- Change Role Label EKS Node ---------------------------------*/
	if destroy == "false" {
		// List all nodes in the cluster
		nodes, err := clientset.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
		if err != nil {
			log.Fatal(err)
		}

		// Label each node with the desired label
		for _, node := range nodes.Items {
			nodeName := node.ObjectMeta.Name
			labels := node.ObjectMeta.Labels
			if labels == nil {
				labels = make(map[string]string)
			}

			// Add or update the label "node-role.kubernetes.io/worker" to "worker"
			labels["node-role.kubernetes.io/worker"] = "worker"

			node.ObjectMeta.Labels = labels
			_, err = clientset.CoreV1().Nodes().Update(context.Background(), &node, metav1.UpdateOptions{})
			if err != nil {
				log.Printf("❌ Failed to label node %s: %v", nodeName, err)

			} else {
				log.Printf("✅ Successfully labeled node %s", nodeName)

			}
		}
	}
	/*--------------------------- Change Role Label EKS Node ---------------------------------*/

	/*--------------------------- Created a Role for EBS CSI Storage ------------------------*/

	//Set Federated, Auth and Sub Trust Relationships For Role at CSI Drivers
	Fed := fmt.Sprintf("%s%s%s%s%s%s", "arn:aws:iam::", AppConfig1.Account, ":oidc-provider/oidc.eks.", AppConfig1.Region, ".amazonaws.com/id/", OpenID)
	Aud := fmt.Sprintf("%s%s%s%s%s", "oidc.eks.", AppConfig1.Region, ".amazonaws.com/id/", OpenID, ":aud")
	Sub := fmt.Sprintf("%s%s%s%s%s", "oidc.eks.", AppConfig1.Region, ".amazonaws.com/id/", OpenID, ":sub")

	// Create a PolicyDocument for the AssumeRolePolicyDocument for CSI Role
	assumeRolePolicy := awsiam.NewPolicyDocument(&awsiam.PolicyDocumentProps{
		Statements: &[]awsiam.PolicyStatement{
			awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
				Effect:  awsiam.Effect_ALLOW,
				Actions: &[]*string{jsii.String("sts:AssumeRoleWithWebIdentity")},
				Principals: &[]awsiam.IPrincipal{
					awsiam.NewFederatedPrincipal(&Fed, nil, nil),
				},
				Conditions: &map[string]interface{}{
					"StringEquals": map[string]interface{}{
						Aud: "sts.amazonaws.com",
						Sub: "system:serviceaccount:kube-system:ebs-csi-controller-sa",
					},
				},
			}),
		},
	})

	// Create a CfnRole with the AssumeRolePolicyDocument
	cfnRole := awsiam.NewCfnRole(stack, &EbsRole, &awsiam.CfnRoleProps{
		AssumeRolePolicyDocument: assumeRolePolicy,
		RoleName:                 &EbsRole,
		ManagedPolicyArns: &[]*string{
			&policyArn,
		},
	})

	/*--------------------- End Created a Role dor EBS CSI Storage ------------------------*/

	// Create an EKS addon using CfnAddon
	EksAddon := awseks.NewCfnAddon(stack, jsii.String("EbsCsiAddon"), &awseks.CfnAddonProps{
		ClusterName:           &clusterName,
		AddonName:             jsii.String("aws-ebs-csi-driver"),
		AddonVersion:          &AppConfig.AddonVersion,
		ServiceAccountRoleArn: cfnRole.AttrArn(),
	})

	EksAddon.Node().AddDependency(cfnRole)

	// Create Storage Class :  managed-csi
	if destroy == "false" {
		scYAMLPath := AppConfig.ScNamef

		scYAML, err := os.ReadFile(scYAMLPath)
		if err != nil {
			fmt.Printf("Error reading SC YAML file: %v\n", err)
			os.Exit(1)
		}

		err = applyResourcesFromYAML(scYAML, clientset, dd)
		if err != nil {
			log.Fatalf("❌ Error applying sc.yaml file: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✅ Storage Class created successfully")
	}

	return stack
}

func main() {
	defer jsii.Close()

	// Read configuration from config.json file
	var configcrd ConfAuth
	var config1 Configuration
	var AppConfig1, AppConfig = GetConfig(configcrd, config1)

	Stack := "EksStackConfig" + AppConfig1.Index
	app := awscdk.NewApp(nil)

	destroy := app.Node().TryGetContext(jsii.String("destroy"))
	destroyStr := destroy.(string)
	if destroy == "true" {
		//Are you sure you want to delete: DevopsStack02 (y/n)? y
		kubeconfigPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
		config, err := rest.InClusterConfig()
		if err != nil {
			config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		}

		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			glog.Fatalf("❌ Failed to create a ClientSet: %v. Exiting.", err)
		}

		storageClassName := AppConfig.ScName

		err = clientset.StorageV1().StorageClasses().Delete(context.TODO(), storageClassName, metav1.DeleteOptions{})
		if err != nil {
			fmt.Printf("❌ Error deleting StorageClass: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✅ StorageClass %s deleted successfully\n", storageClassName)

	}

	NewEksstackconfigStack(app, Stack, &EksstackconfigStackProps{
		awscdk.StackProps{
			Env: env(AppConfig1.Region, AppConfig1.Account),
		},
	}, AppConfig, AppConfig1, destroyStr)

	app.Synth(nil)
}

func env(Region1 string, Account1 string) *awscdk.Environment {

	return &awscdk.Environment{
		Account: &Account1,
		Region:  &Region1,
	}
}
