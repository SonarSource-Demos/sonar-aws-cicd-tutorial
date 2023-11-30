package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awseks"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
	kubectlv27 "github.com/cdklabs/awscdk-kubectl-go/kubectlv27/v2"
)

type EksStackProps struct {
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

func GetConfig(configcrd ConfAuth, configjs Configuration) (ConfAuth, Configuration) {

	fconfig, err := os.ReadFile("config.json")
	if err != nil {
		panic("❌ Problem with the configuration file : config.json")
		os.Exit(1)
	}
	if err := json.Unmarshal(fconfig, &configjs); err != nil {
		fmt.Println("❌ Error unmarshaling JSON:", err)
		os.Exit(1)
	}

	fconfig2, err := os.ReadFile("../config_crd.json")
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

func NewEksStack(scope constructs.Construct, id string, props *EksStackProps, AppConfig Configuration, AppConfig1 ConfAuth) awscdk.Stack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)

	// Set Variables
	var clusterName = AppConfig.ClusterName + AppConfig1.Index
	var AdmRole = clusterName + AppConfig.EksAdminRole

	// ARN policies for Role eksadmin
	var policyArn2 = "arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy"
	var policyArn3 = "arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy"
	var policyArn4 = "arn:aws:iam::aws:policy/ElasticLoadBalancingFullAccess"
	var policyArn5 = "arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly"
	var policyArn6 = "arn:aws:iam::aws:policy/CloudWatchAgentServerPolicy"
	var policyArn7 = "arn:aws:iam::aws:policy/AmazonEC2FullAccess"
	var policyArn8 = "arn:aws:iam::aws:policy/AmazonEKSVPCResourceController"
	var policyArn9 = "arn:aws:iam::aws:policy/AmazonEKSClusterPolicy"

	// Open AWS session
	sess := session.Must(session.NewSession(&aws.Config{
		Region: &AppConfig1.Region,
	}))

	//------------------------Get Sts Account --------------------------------------//
	// Create an STS service client
	svc := sts.New(sess)

	// Create an STS API request to get caller identity
	inputuser := &sts.GetCallerIdentityInput{}
	resultuser, err := svc.GetCallerIdentity(inputuser)

	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	// Access and print the caller's ARN
	ArnPrincipal := *resultuser.Arn

	//------------------------END Get Sts Account --------------------------------------//

	// Get VPC and Set Variables for EC2 instance
	PartVpc := awsec2.Vpc_FromLookup(stack, &AppConfig.VPCid, &awsec2.VpcLookupOptions{VpcId: &AppConfig.VPCid})

	Instance := AppConfig.Instance
	InstanceSZ := AppConfig.InstanceSize

	// Define the trusted service principals dor EKS RoleAdmin
	trustedService1 := awsiam.NewServicePrincipal(jsii.String("eks.amazonaws.com"), nil)
	trustedService2 := awsiam.NewArnPrincipal(&ArnPrincipal)
	trustedPrincipals := awsiam.NewCompositePrincipal(trustedService1, trustedService2)

	// Define an IAM policy statement with multiple actions
	myPolicyStatement := awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Effect: awsiam.Effect_ALLOW,
		Actions: &[]*string{
			jsii.String("ec2:CreateVolume"),
			jsii.String("ec2:DeleteVolume"),
			jsii.String("ec2:DetachVolume"),
			jsii.String("ec2:AttachVolume"),
			jsii.String("ec2:DescribeInstances"),
			jsii.String("ec2:CreateTags"),
			jsii.String("ec2:DeleteTags"),
			jsii.String("ec2:DescribeTags"),
			jsii.String("ec2:DescribeVolumes"),
		},
		Resources: &[]*string{
			jsii.String("*"),
		},
	})

	// Define IAM role for the EKS cluster.
	eksAdminRole := awsiam.NewRole(stack, &AdmRole, &awsiam.RoleProps{
		AssumedBy: trustedPrincipals,
		RoleName:  &AdmRole,
	})

	eksAdminRole.AddManagedPolicy(awsiam.ManagedPolicy_FromManagedPolicyArn(stack, jsii.String("AmazonEKSWorkerNodePolicy"), &policyArn2))
	eksAdminRole.AddManagedPolicy(awsiam.ManagedPolicy_FromManagedPolicyArn(stack, jsii.String("AmazonEKS_CNI_Policy"), &policyArn3))
	eksAdminRole.AddManagedPolicy(awsiam.ManagedPolicy_FromManagedPolicyArn(stack, jsii.String("ElasticLoadBalancingFullAccess"), &policyArn4))
	eksAdminRole.AddManagedPolicy(awsiam.ManagedPolicy_FromManagedPolicyArn(stack, jsii.String("AmazonEC2ContainerRegistryReadOnly"), &policyArn5))
	eksAdminRole.AddManagedPolicy(awsiam.ManagedPolicy_FromManagedPolicyArn(stack, jsii.String("CloudWatchAgentServerPolicy"), &policyArn6))
	eksAdminRole.AddManagedPolicy(awsiam.ManagedPolicy_FromManagedPolicyArn(stack, jsii.String("AmazonEC2FullAccess"), &policyArn7))
	eksAdminRole.AddManagedPolicy(awsiam.ManagedPolicy_FromManagedPolicyArn(stack, jsii.String("AmazonEKSVPCResourceController"), &policyArn8))
	eksAdminRole.AddManagedPolicy(awsiam.ManagedPolicy_FromManagedPolicyArn(stack, jsii.String("AmazonEKSClusterPolicy"), &policyArn9))
	eksAdminRole.AddToPolicy(myPolicyStatement)

	// Create the EKS cluster.
	eksCluster := awseks.NewCluster(stack, &clusterName, &awseks.ClusterProps{
		ClusterName:             &clusterName,
		Vpc:                     PartVpc,
		Role:                    eksAdminRole,
		MastersRole:             eksAdminRole,
		Version:                 awseks.KubernetesVersion_Of(&AppConfig.K8sVersion),
		KubectlLayer:            kubectlv27.NewKubectlV27Layer(stack, jsii.String("kubectl127layer")),
		DefaultCapacity:         &AppConfig.Workernode,
		DefaultCapacityInstance: awsec2.InstanceType_Of(awsec2.InstanceClass(Instance), awsec2.InstanceSize(InstanceSZ)),
		DefaultCapacityType:     awseks.DefaultCapacityType_NODEGROUP,
		EndpointAccess:          awseks.EndpointAccess_PUBLIC(),
		OutputConfigCommand:     jsii.Bool(true),
		Tags: &map[string]*string{
			"Env":                               jsii.String("Dev"),
			"k8s.io/cluster-autoscaler/enabled": jsii.String("true"),
		},
		AlbController: &awseks.AlbControllerOptions{
			Version: awseks.AlbControllerVersion_V2_5_1(),
		},
	})

	//Add Dependency : waiting The Adim Role created
	eksCluster.Node().AddDependency(eksAdminRole)

	// Output the EKS cluster name.
	awscdk.NewCfnOutput(stack, jsii.String("EksClusterName"), &awscdk.CfnOutputProps{
		Value: eksCluster.ClusterName(),
	})

	return stack
}

func main() {
	defer jsii.Close()

	// Read configuration from config.json file
	var configcrd ConfAuth
	var config1 Configuration
	var AppConfig1, AppConfig = GetConfig(configcrd, config1)
	Stack1 := "EksStack" + AppConfig1.Index

	app := awscdk.NewApp(nil)

	NewEksStack(app, Stack1, &EksStackProps{
		awscdk.StackProps{
			Env: env(AppConfig1.Region, AppConfig1.Account),
		},
	}, AppConfig, AppConfig1)

	app.Synth(nil)

}

func env(Region1 string, Account1 string) *awscdk.Environment {

	return &awscdk.Environment{
		Account: &Account1,
		Region:  &Region1,
	}
}
