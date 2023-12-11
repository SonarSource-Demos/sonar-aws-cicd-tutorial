# Deploying SonarQube to EKS

In this chatper we'll deploy SonarQube to a new EKS Cluster

## Prerequisites

### SonarQube license

The Developer Edition we deploy in the present chapter requires a license.
If you don't have one, please reach out [here](https://www.sonarsource.com/plans-and-pricing/developer/) to  obtain one (a Developer Edition 100k is sufficient for the present tutorial).
As an alternative, you may go over the SonarQube deployment to learn about it but use [SonarCloud](https://www.sonarsource.com/products/sonarcloud/), which is free for open-source (public) analysis, in the CI/CD steps of the tutorial.

A Developer Edition SonarQube instance without a license will allow you to run all the tutorial steps until the first pipeline analysis (which will fail).

### Important Sonar resources

The SonarQube documentation is always good to have as a bookmark: https://docs.sonarsource.com/sonarqube/

In this chapter we'll deploy SonarQube to EKS, using the [SonarQube generic Helm Chart](https://github.com/SonarSource/helm-chart-sonarqube/). This will simplify configuration of the deployment and you can rely on the built-in best practices maintained by the Sonar team.

### Deployment tools

It's recommended to run this tutorial from your local machine and also learn about [SonarLint](https://www.sonarsource.com/products/sonarlint/). Most of these tools are common tools used by DevOps engineers working on AWS, some of them would be used by Developers hosting their projects on CodeCommit. Your local machine will need the following:

* IntelliJ or VisualStudio Code IDE
* SonarLint plugin installed from the IDE marketplace
* [AWS CLI](https://aws.amazon.com/cli/)
* [Git CLI](https://git-scm.com/downloads)
* [AWS git-remote-codecommit](https://docs.aws.amazon.com/codecommit/latest/userguide/setting-up-git-remote-codecommit.html) Git extension
* [kubectl](https://docs.aws.amazon.com/eks/latest/userguide/install-kubectl.html)
* [helm](https://helm.sh/docs/intro/install/)
* [NodeJS](https://nodejs.org/en/download)
* [eksctl](https://eksctl.io/installation/)
* [Go](https://go.dev/doc/install)
* [AWS CDK](https://docs.aws.amazon.com/cdk/v2/guide/getting_started.html)

As an alternative, you may be able to run most of the present tutorial steps from Cloud9.

Make sur you're authenticated from the command line on your AWS IAM profile. You may use the following command line to validate your credentials and your connectivity.

```bash
aws sts get-caller-identity
```

## Cluster creation

AWS Cloud Development Kit will be used to setup your AWS environement and various resources.
The [cdk folder](./cdk) in this repository hold several scripts that will help you throughout your EKS stack setup.

### Step 1 - Set your profile

In this step, you'll configure your AWS profiles for the following cdk scripts.

```bash
cd cdk
```

And edit the [config_crd.json](../../cdk/config_crd.json) according to your aws credentials and preferences:

* Region: Deployment region
* Account: AWS account number
* SSOProfile: AWS SSO Profile
* Index: appended to the names of your VPC, EKS Cluster, AWS Secret, Stacks... to avoid any conflict if you need to iterate or share an account
* AWSsecret: AWS Secret name for SonarQube

### Step 2 - Set your VPC and Security Groups

The step will configure your Virtual Private Network to host SonarQube (and later your Java app) and set the permissions for your resources.

*Note: If you need to reuse an existing VPC, you may skip this step. In that case, please make sure your private subnet has the following tag: ```kubernetes.io/role/internal-elb=1``` and that your public subnet has this one: ```kubernetes.io/role/elb=1```*

This step is run form the [vpc](../../cdk/vpc) folder:

```bash
cd vpc
```

* The ```cdk.json``` file in the folder describes how the toolkit should execute the VPC creation. You should not need to edit that file.
* the ```config.json``` file must be configured for your setup:
  * VPCName: Name for your new VPC (will get suffixed with your index)
  * VPCcidr: [CIDR blocks](https://docs.aws.amazon.com/vpc/latest/userguide/vpc-cidr-blocks.html)
  * ZA:	Number of Availability zones (minimum 2)
  * SGName: Security Group Name
  * SGDescription: Security Group Desciption

Once it's done, run the following commands in the vpc folder:
```bash
# Install the required go modules based on the go.mod and go.sum files
go mod download
# check your changes
cdk diff
# Create the VPC
cdk deploy
```

The last command should output your new stack Name, ID, and ARN, e.g.:

```text
VPCStack01: deploying... [1/1]
VPCStack01: creating CloudFormation changeset...

 ✅  VPCStack01

✨  Deployment time: 199.91s

Outputs:
VPCStack01.VPCCREATED = vpc-09984b9cc1c290321
Stack ARN:
arn:aws:cloudformation:eu-central-1:108878442956:stack/VPCStack01/37a5c9d0-78c0-11ee-2503-02749d6c9b37

✨  Total time: 206.25s
```

### Step 3 - Create your EKS cluster

This step will create your new EKS cluster. the CDK commands and config files are in the [eks](../../cdk/eks/) folder.

```bash
cd ../eks
```

* The ```cdk.json``` file in the folder describes how the toolkit should execute the cluser creation. You don't need to edit that file.
* the ```config.json``` file must be configured for your setup (the provided values should work except for the VPC ID which you MUST change with your own)
  * ClusterName: EKS Cluster Name
  * VPCid: ID of the VPC created above
  * K8sVersion: Version of Kubernetes: default 1.27
  * Workernode: Number of Worker Nodes (e.g. 2)
  * EksAdminRole:  Name of the EKS admin role
  * EBSRole: Name of the EBS Role for storage
  * Instance: AWS Instance types using for EKS (Arm-based instances not supported by the database)
  * InstanceSize: AWS Instance size
  * AddonVersion: Addon version for EBS CSI Driver : 1.24.0-eksbuild.1
  * ScName: Name of the Storage Class
  * ScNamef: Path of store class manifest file for addons

Once it's done, run the following commands in the eks folder:

```bash
# Install the required go modules based on the go.mod and go.sum files
go mod download
# check your changes
cdk diff
# Create the VPC
cdk deploy
```

The command will require a few minutes to run and should output the details of your new cluster:

```text
 ✨  Deployment time: 1018.37s

Outputs:
✅  EksStack02.SonarAWSTutoConfigCommandFAA0F346 = aws eks update-kubeconfig --name SonarAWSTuto --region eu-central-1 --role-arn arn:aws:iam::XXXXXX:role/SonarAWSTuto-02-AdminRole

✨  Total time: 1023.36s
```

You need to apply changes to your kubectl credentials accordingly. Copy the proposed kubectl command, and replace the XXXXXX part with your own Account ID:

```bash
aws eks update-kubeconfig --name SonarAWSTuto-02 --region eu-central-1 --role-arn arn:aws:iam::XXXXXX:role/SonarAWSTuto-02-AdminRole
```

You may check your cluster is running with your configured number of nodes:

```bash
kubectl get nodes
```

```text
NAME                                            STATUS   ROLES    AGE   VERSION
ip-192-168-189-143.eu-central-1.compute.internal   Ready    <none>    5m   v1.27.7-eks-e71965b
ip-192-168-240-14.eu-central-1.compute.internal    Ready    <none>    4m   v1.27.7-eks-e71965b
```

### Step 4 - Storage Class addons

The present tutorial requires the EBS CSI Driver, which we will install as [an EKS add-on](https://docs.aws.amazon.com/eks/latest/userguide/managing-ebs-csi.html). For this steps, the script will run directly using Go:

```bash
cd addons
# Install the required go modules based on the go.mod and go.sum files
go mod download
# deploy the EBS CSI driver
cdk deploy --context destroy=false
```

You may check it was activated on the kube-system namespace of your cluster

```bash
kubectl get deployment/ebs-csi-controller -n kube-system
```

```text
NAME                 READY   UP-TO-DATE   AVAILABLE   AGE
ebs-csi-controller   2/2     2            2           1m00s
```

And if the managed-csi storage class is indeed available:

```bash
kubectl get sc
```

```text
NAME            PROVISIONER             RECLAIMPOLICY   VOLUMEBINDINGMODE      ALLOWVOLUMEEXPANSION   AGE
gp2 (default)   kubernetes.io/aws-ebs   Delete          WaitForFirstConsumer   false                  12h
managed-csi     ebs.csi.aws.com         Delete          WaitForFirstConsumer   false                  1m56s
```

### Step 4 - SonarQube Helm deployment

In this step, we'll deploy SonarQube using its [official Helm Chart](https://SonarSource.github.io/helm-chart-sonarqube), the only Sonar supported way to deploy SonarQube to a Kubernetes cluster.

Let's make our [preferred region](../../cdk/config_crd.json) our default one:

```bash
export AWS_DEFAULT_REGION="eu-central-1"
```

1. Add the SonarQube Helm Chart repository:

```bash
# let's user the sonarqube deployment folder
cd ../../../sonarqube
# and fetch the latest Helm Chart from the official repository
helm repo add sonarqube https://SonarSource.github.io/helm-chart-sonarqube
helm repo update
```

2. As [documented with the HelmChart](https://github.com/SonarSource/helm-chart-sonarqube/tree/master/charts/sonarqube), we'll configure the service by providing a specific `values.yaml` file. A ready-to-go file is proposed in the current folder.
3. Use helm to deploy your first release of the service to the new EKS cluster. As per [helm documentation](https://helm.sh/docs/topics/architecture/) *a release is a running instance of a chart, combined with a specific config*, we'll name our initial release `sonarqube-release`:

```bash
helm upgrade --install -f ./values.yaml  --create-namespace -n sonarqube sonarqube-release sonarqube/sonarqube
```

4. Check that SonarQube instance is deployed successfully:

```bash
kubectl get all -n sonarqube
```

```text
NAME                                             READY   STATUS    RESTARTS   AGE
pod/sonarqube-release-postgresql-0                 1/1     Running   0          5d7h
pod/sonarqube-release-sonarqube-6464dc554f-m5nv8   1/1     Running   0          5d7h

NAME                                            TYPE           CLUSTER-IP    EXTERNAL-IP                     PORT(S)        AGE
service/sonarqube-release-postgresql            ClusterIP      X.X.X.X        <none>                         5432/TCP       5d7h
service/sonarqube-release-postgresql-headless   ClusterIP      None           <none>                         5432/TCP       5d7h
service/sonarqube-release-sonarqube             LoadBalancer   X.X.X.X    XX.eu-central-1.elb.amazonaws.com  9000:32267/TCP  5d7h

NAME                                          READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/sonarqube-release-sonarqube   1/1     1            1           5d7h

NAME                                                    DESIRED   CURRENT   READY   AGE
replicaset.apps/sonarqube-release-sonarqube-6464dc554f   1         1         1       5d7h

NAME                                            READY   AGE
statefulset.apps/sonarqube-release-postgresql   1/1     5d7h
```

You may also check that all ressources and services were correctly attached to the cluster:

```bash
kubectl get po,svc,pv -n kube-system
```

```text
NAME                                                READY   STATUS    RESTARTS   AGE
pod/aws-load-balancer-controller-5b9c68895c-gzvl2   1/1     Running   0          43m
pod/aws-load-balancer-controller-5b9c68895c-xlzr4   1/1     Running   0          43m
pod/aws-node-2vx5f                                  1/1     Running   0          44m
pod/aws-node-2z6d9                                  1/1     Running   0          44m
pod/coredns-577fccf48c-bhjsq                        1/1     Running   0          51m
pod/coredns-577fccf48c-kcfg8                        1/1     Running   0          51m
pod/ebs-csi-controller-7f4b7f59c7-s2nrj             6/6     Running   0          24m
pod/ebs-csi-controller-7f4b7f59c7-tmn47             6/6     Running   0          24m
pod/ebs-csi-node-rvchj                              3/3     Running   0          24m
pod/ebs-csi-node-vhksl                              3/3     Running   0          24m
pod/kube-proxy-k7qrg                                1/1     Running   0          44m
pod/kube-proxy-ncl2m                                1/1     Running   0          44m

NAME                                        TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)         AGE
service/aws-load-balancer-webhook-service   ClusterIP   10.100.193.65   <none>        443/TCP         43m
service/kube-dns                            ClusterIP   10.100.0.10     <none>        53/UDP,53/TCP   51m

NAME                                                        CAPACITY   ACCESS MODES   RECLAIM POLICY   STATUS   CLAIM                                           STORAGECLASS   REASON   AGE
persistentvolume/pvc-24c2ab5e-5221-4605-b93c-847e87e4669f   5Gi        RWO            Delete           Bound    sonarqube/data-sonarqube-release-postgresql-0   gp2                     15m
persistentvolume/pvc-c0d209a5-cfcf-4cbd-88ee-fba086f2e657   5Gi        RWO            Delete           Bound    sonarqube/sonarqube-release-sonarqube
```

> ❗️ **Warning:**
> This Sonar deployment serves us well for this tutorial but is **NOT** suitable for production:
> * Performance bootstrap checks are disabled❗️
> * Clear text credentials in helm config file (secrets not managed)❗️
> * PosgreSQL DB deployed in the Applciative cluster (you may want to leverage an AWS DB instead)❗️
> * No Ingress or RP is configured to encrypt the traffic over https❗️

### Step 5 - Collect SonarQube public address

You may check the cluster events, the pod status and the SonarQube logs to see the SonarQube service starting:

```bash
 kubectl get events -n sonarqube -w
 kubectl get pods -n sonarqube
 kubectl logs service/sonarqube-release-sonarqube -n sonarqube
 ```

Once SonarQube Service is ready, you can check the public address assigned to it, with this command:

```bash
kubectl get svc -w sonarqube-release-sonarqube -n sonarqube
```

```text
NAME                          TYPE           CLUSTER-IP       EXTERNAL-IP                                                                     PORT(S)          AGE
sonarqube-release-sonarqube   LoadBalancer   10.100.178.215   k8s-sonarqub-sonarqub-2ac585b2a4-58216b37bcec022f.elb.eu-central-1.amazonaws.com   9000:32440/TCP   2m38s
```

> ✨Note: You may uninstall this SonarQube deployment, your data will be preserved as long as the Persistent Volume storing the PostgreSQL DB is not lost.

-----
[Previous](../1.SonarCleanCode/README.md) | [Next](../3.ConfigureSonarQube/README.md)
