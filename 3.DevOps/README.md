# AWS CI/CD

In this section, we'll prepare the example application repository and the pipeline that will build, analyze, and deploy this application.
The entire stack makes use of:

* CodeCommit: to host the repository
* CodeBuild: to run your pipeline
* CodePipeline: to trigger the pipeline on events
* EventBridge: to allow builds to run on Pull Request events
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

```bash
# Install the required go modules based on the go.mod and go.sum files
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
As this script leverages your aws SSO credentials, you MUST be logged with it before you can use it.

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

## EventBridge for PR analysis

In a fast-moving software development process, we want to get feedback on code quality as early as possible. Traditionally this would require a commit to the delivery branch and waiting for the pipeline to complete. By using Sonar's Pull Request (PR) analysis feature we can shorten the feedback loop protect the delivery branches from 'bad code'.

The `AWS Sonar Plugin` is a CloudFormation template that deploys an Amazon EventBridge rule to enable the previously described workflow.

### How does the AWS Sonar Eventbridge Plugin work?

The Plugin uses EventBridges input transformers to bring the event data into the right format and then triggers CodeBuilds StartBuild API. The plugin covers the highlighted "pre-merge validation" phase:

![EventBridge schema](/assets/3.DevOps/multibranch-PR-flow.png)

### Deploy the CloudFormation template

In this step you will deploy the CloudFormation script: [eventbridge-rule-codebuild.json](/assets/3.DevOps/eventbridge-rule-codebuild.json)
Before we may deploy it we need to take note of tsome wo important values of your CI/CD infrastructure:

1. First run the following command to get your repository identifier:

```bash
# list the repositories
aws codecommit list-repositories
# get your repository details
aws codecommit get-repository --repository-name sonar-sample-app-02
```

Take note of the returned `Arn`

2. Then we'll need your codebuild identifier:

```bash
# list your build projects:
aws codebuild list-projects
# get the identifier for your project
aws codebuild batch-get-projects --names clean-java-code-build-02
```

Take note of the returned `arn` value

3. Now Navigate to the ClouFormation Console for your zone (e.g. [here](https://eu-west-1.console.aws.amazon.com/cloudformation/home) for eu-west-1)
1. Select `Create stack` and `With new resources` (standard)
2. Upload the eventbridge-rule-codebuild.json and select `Next`
3. Give the stack a name like *sonar-eventbridge-plugin*
4. Enter the CodeCommitRepositoryARN and SonarCodeBuildProjectARN you previously took note of.
5. Click `Next`
6. Check `I acknowledge that AWS CloudFormation might create IAM resources` and yhen `Submit`

This will take a minute to deploy.

### Variables in buildspec.yaml

The pipeline definition file, `buildspec.yml`, is an important piece of the puzzle. It is triggered for all types of builds, main branch builds **and** PR analysis builds through CodePipeline. It reads all of the previously set variables and needs to understand how to trigger the [SonarScanner for Maven](https://docs.sonarsource.com/sonarqube/9.8/analyzing-source-code/scanners/sonarscanner-for-maven/).

Take a look at your buildspec.yml at the root of your CodeCommit repository .

In detail, the buildspec.yml needs to declare and use the following variables to be compatible with the AWS Sonar Plugin:

* `SourceBranch` for the name of the feature branch to be merged (PR cases) or the name of the long-lived branch (usually main)
* `DestinationBranch` (only available if the build is triggered by a PR!) for the branch the PR targets
* `PRKey` (only available if the build is triggered by a PR!) the ID of the PR. CodeCommit manages these ID.

Everything is now ready for PR and branch analysis.

----
[Previous](../2.CleanCode/3.ConfigureSonarQube/README.md) | [Next](../4.DevWorkflow/README.md)