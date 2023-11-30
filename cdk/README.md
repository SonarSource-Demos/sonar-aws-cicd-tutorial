 ![SonarQube](images1/sonar.png)![Amazon ECS](https://img.shields.io/static/v1?style=for-the-badge&message=Amazon+ECS&color=222222&logo=Amazon+ECS&logoColor=FF9900&label=)![Static Badge](https://img.shields.io/badge/Go-v1.21-blue:) ![Static Badge](https://img.shields.io/badge/AWS_CDK-v2.96.2-blue:)

The purpose of this tutorial is to guide you through the various steps involved in deploying sonarqube and a java application in an AWS EKS environment using AWS Cloud Development Kit (AWS CDK) for golang.
The java application will be stored in AWS CodeCommit and will use CodeBuild integrated with sonarqube (for code analysis) for production release 

The AWS CDK lets you build reliable, scalable, cost-effective applications in the cloud with the considerable expressive power of a programming language.
A CloudFormation template is generated for each deployment.

![Flow CDK](images1/diagramcdk.png)



## Prerequisites

Before you get started, you’ll need to have these things:

* AWS account
* SSO Login
* [AWS CLI V2](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html)
* [AWS Cloud Development Kit (AWS CDK) v2](https://docs.aws.amazon.com/cdk/v2/guide/getting_started.html)
* [Go language installed](https://go.dev/)
* [Node.jjs installed](https://nodejs.org/en)
* [Kubectl installed](https://docs.aws.amazon.com/eks/latest/userguide/install-kubectl.html) is a command line tool that you use to communicate with the Kubernetes API 
server.
* A Git Client
* [AWS git-remote-codecommit](https://docs.aws.amazon.com/codecommit/latest/userguide/setting-up-git-remote-codecommit.html) Git extension. This will be used to authenticate requests to the repo
* [eksctl installed](https://eksctl.io/installation/) 

When setting up a new AWS environment for our project, one of the first things you'll need to do is create a VPC.
When setting up the VPC, it is essential to configure security groups to control inbound and outbound traffic to and from the VPC. Security groups act as virtual firewalls, allowing only authorized traffic to pass through.
The ports to be authorized (defined in the Security Groups) for input/output are : 9000 (sonarqube default port)

## Steps

### ✅ Set Config AWS Profil

The `config_crd.json` Contains the parameters to be initialized to AWS Profil 

```
config_crd.json :

Region:  Deployment region	        
Account: AWS account number
SSOProfile: AWS SSO Profile using
Index: Number to generate a name for the VPC, EKS Cluster,AWS Secret, Stacks .... : <NAME+INDEX>
AWSsecret: AWS Secret name for sonarqube 
```    
❗️ You must initialize these variables with your informations.

### ✅ Creating a VPC

If you already have VPC to create you can skip this step.</br>
✅ Before deploying your EKS cluster you must check that your 
Private subnet has the tag: 
* ✨ kubernetes.io/role/internal-elb=1 

and you Public subnet the tag:
* ✨ kubernetes.io/role/elb=1

Please see [subnet requirements and considerations](https://docs.aws.amazon.com/eks/latest/userguide/network_reqs.html)

go to directory [VPC](https://github.com/SonarSource-Demos/aws-workshop/tree/sonar-browsable-markdown/static/assets/CDK/VPC3) (please read the README.md)

### ✅ Creating a EKS Cluster
go to directory [EKS](https://github.com/SonarSource-Demos/aws-workshop/tree/sonar-browsable-markdown/static/assets/CDK/EKS) (please read the README.md)

### ✅ SonarQube deployment
go to directory [sonarqube](https://github.com/SonarSource-Demos/aws-workshop/tree/sonar-browsable-markdown/static/assets/CDK/sonarqube) (please read the README.md)

## 📛 This repos is in development 📛

## ✅ Ressources

▶️ [awscdk go package](https://pkg.go.dev/github.com/aws/aws-cdk-go/awscdk/v2#section-readme) 

▶️ [awseks go package](https://pkg.go.dev/github.com/aws/aws-cdk-go/awscdk/v2/awseks#section-documentation)

▶️ [awsiam go package](https://pkg.go.dev/github.com/aws/aws-cdk-go/awscdk/v2@v2.102.0/awsiam#section-readme)

▶️ [awsec2 go package](https://pkg.go.dev/github.com/aws/aws-cdk-go/awscdk/v2/awsec2#section-readme)

▶️ [Amazon EKS VPC and subnet requirements and considerations](https://docs.aws.amazon.com/eks/latest/userguide/network_reqs.html)