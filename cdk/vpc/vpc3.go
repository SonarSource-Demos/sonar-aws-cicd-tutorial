package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type Vpc3StackProps struct {
	awscdk.StackProps
}
type ConfAuth struct {
	Region     string
	Account    string
	SSOProfile string
	Index      string
}

type Configuration struct {
	VpcName       string
	Vpccidr       string
	Za            float64
	SgName        string
	SgDescription string
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

func NewVpc3Stack(scope constructs.Construct, id string, props *Vpc3StackProps, AppConfig Configuration, AppConfig1 ConfAuth) awscdk.Stack {

	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)
	var vpcName = AppConfig.VpcName + AppConfig1.Index
	var SGName = AppConfig.SgName + AppConfig1.Index

	// Open AWS session
	sess := session.Must(session.NewSession(&aws.Config{
		Region: &AppConfig1.Region,
	}))

	tagProps := &awscdk.TagProps{
		ApplyToLaunchedInstances: jsii.Bool(false),
		Priority:                 jsii.Number(123),
	}

	// Get VPCID by VPCName for testing if VPC exist
	svc1 := ec2.New(sess)

	vpcimport, err := svc1.DescribeVpcs(&ec2.DescribeVpcsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("tag:Name"),
				Values: []*string{&vpcName},
			},
		},
	})

	if err != nil {
		fmt.Println("❌ Error listing VPC:", err)
		os.Exit(1)
	}

	size := len(vpcimport.Vpcs)

	if size == 0 {
		// Create a new VPC
		// Define the VPC with IPv4 CIDR block.
		vpc := awsec2.NewVpc(stack, &vpcName, &awsec2.VpcProps{
			IpAddresses: awsec2.IpAddresses_Cidr(&AppConfig.Vpccidr),
			MaxAzs:      &AppConfig.Za,
			VpcName:     &vpcName,
		})

		// Create a security group within the VPC.
		securityGroup := awsec2.NewSecurityGroup(stack, &SGName, &awsec2.SecurityGroupProps{
			Vpc:               vpc,
			SecurityGroupName: &SGName,
			Description:       &AppConfig.SgDescription,
		})

		securityGroup.Node().AddDependency(vpc)
		// Add ingress and egress rules to the security group.
		securityGroup.AddEgressRule(awsec2.Peer_AnyIpv4(), awsec2.Port_AllTraffic(), jsii.String("Allow all outbound traffic"), jsii.Bool(true))

		// Tags Subnets for  to be used by EKS

		for _, subnet := range *vpc.PublicSubnets() {
			awscdk.Tags_Of(subnet).Add(jsii.String("kubernetes.io/role/elb"), jsii.String("1"), tagProps)
		}

		for _, subnet := range *vpc.PrivateSubnets() {
			awscdk.Tags_Of(subnet).Add(jsii.String("kubernetes.io/role/internal-elb"), jsii.String("1"), tagProps)
		}

		awscdk.NewCfnOutput(stack, aws.String("VPC_CREATED"), &awscdk.CfnOutputProps{
			Description: aws.String("The VPC Created"),
			Value:       vpc.VpcId(),
		})

	} else {
		awscdk.NewCfnOutput(stack, aws.String("VPC_EXIST"), &awscdk.CfnOutputProps{
			Description: aws.String("The VPC already exists"),
			Value:       vpcimport.Vpcs[0].VpcId,
		})
	}

	return stack
}

func main() {
	defer jsii.Close()

	// Read configuration from Config.json file
	var configcrd ConfAuth
	var config1 Configuration
	var AppConfig1, AppConfig = GetConfig(configcrd, config1)

	Stack := "VPCStack" + AppConfig1.Index

	app := awscdk.NewApp(nil)

	NewVpc3Stack(app, Stack, &Vpc3StackProps{
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
