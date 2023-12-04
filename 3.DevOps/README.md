# AWS CI/CD

In this section, we'll prepare the example repository and the pipeline that will build, analyze, and deploy it.
The entire stack makes use of:

* CodeCommit: to host the repository
* CodeBuild: to run your pepeline
* CodePipeline: to trigger the pipeline on events
* EventBridge: to allow builds to run on Pull Request events, and also on commits on branches that are not the main branch
* Elastic Container Registry (ECR): to host the created Application docker images
* Elastic Kubernetes Services (EKS): to run the application

## CDK Devops Resource creation

Two scripts have been designed to automate the creation and configuration of the entire stack. Here is what they're doing:

* Create the **sonar-sample-app** CodeCommit repository
* Create the repository for the container images
* Define the build configuration with AWS CodeBuild
* Modify IAM permissions for CodeBuild
* Allow CodeBuild to deploy to the EKS cluster
* Setup CodePipeline

This configuration is done from the ```cdk/devops``` folder in the present repository:
```bash
cd cdk/devops
```

* The ```cdk.json``` file in the folder describes how the toolkit should execute resources creation. You should not need to edit that file.
* The ```config.json``` file provide details about them.
  * Reponame: CodeCommit repository name: "sonar-sample-app"
  * Desc: repository description
  * GitRepo: public repository that will get pushed to CodeCommit (as private)
  * Recr: Repository name for container images : app-container-repo
  * ImgTag: Image TAG : Latest
  * BuildPr: Build project name : clean-java-code-build
  * PiplineN: CodePipeline name
  * ClusterName: Set the name of the cluster your created to host SonarQube (without its index)
  * EksAdminRole  AdminRole name

❗️ For everything to work, do not change anything but the cluster name

Once you have set your cluster name in the ```config.json``` file, run the following commands in the devops folder:

# Install the required go modules based on the go.mod and go.sum files
```bash
go mod download
# check your changes
cdk diff
# Create the resources
cdk deploy
```
A successful run will conclude as follows:
```text
 ✅  DevopsStack04

✨  Deployment time: 270.36s

Outputs:
DevopsStack04.ARNRoleBuildProject = arn:aws:iam::123478389876:role/BuildAdminRole04
Stack ARN:
arn:aws:cloudformation:eu-west-3:123478389876:stack/DevopsStack04/236948c0-929c-11ee-80f6-0651b899dbc8

✨  Total time: 276.4s
```

## Populate the repository

Once the repository is created, let's populate it:
As this script leveage your aws SSO credentials, you MUST be logged with it before you can use it.

```bash
# ensure aws sso credentials are set
aws sso login
# populate the repository
go run gitdep.go deploy
```
A successful run will output the following
```text
✅ CodeCommit repository created successful.
✅ Successfully updated EKS Admin Role.
✅ Successfully updated aws-auth ConfigMap.
✅ Clone GitHub App Java Demo is successful.
✅ Commit : M  buildspec.yml
✅ Modify buildspec.yaml is successful.
✅ Push Repository in CodeCommit Repository is successful.
```

**Note**
In case you need to run run the commands in this page multiple times, you might need to manually remove a trust policy added to your cluster admin role:
1. Go to AWS IAM
2. Search for the ```<ClusterName><Index>AdminRole``` role
3. Edit the trust policy
4. Manually remove the 2nd, and last, trusted entity, which look like:

```json
		{
			"Effect": "Allow",
			"Principal": {
				"AWS": "arn:aws:iam::123478389876:role/BuildAdminRole04"
			},
			"Action": "sts:AssumeRole"
		}
```

5. Validate your change with the **Update policy** button

## Validate your setup

On your AWS management console, you can now see your repository
![Repository](/assets/3.DevOps/repository.png)
It was populated:
![Repository Content](/assets/3.DevOps/repository-content.png)
A first build should already run after the code push to the repo:
![CodeBuild](/assets/3.DevOps/codeBuild-started.png)

When the build is successful, you may check its logs for your Java Application Service URL, and check if the service answers as expected:

![JavaApp Running](/assets/3.DevOps/javaApp-running.png)

A new namespace was created in your cluster to run the Application:

```bash
kubectl get namespaces
```

```text
NAME                   STATUS   AGE
default                Active   55m
kube-node-lease        Active   55m
kube-public            Active   55m
kube-system            Active   55m
sonar-aws-javaapp-ns   Active   3m
sonarqube              Active   25m
```

One applicative pod is running within the new namespace:

```bash
kubectl get pods -n sonar-aws-javaapp-ns
````

```text
NAME                                        READY   STATUS    RESTARTS   AGE
sonar-aws-javaapp-deploy-123c4c85b4-h6x45   1/1     Running   0          1m46s
```

----
[Previous](../2.CleanCode/3.ConfigureSonarQube/README.md) | [Next](../4.DevWorkflow/README.md)