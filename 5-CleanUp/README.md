# Clean Up AWS resources

This page should allow you to clean up the tutorial AWS resources so that:

* you may restart "from scratch"
* you stop any resource consumption that would incur some costs

The approach used for cleanup is to clear resources in the reverse order of their creation.

## DevOps stack

From the root folder of the tutorial repository, apply the following commands:

```bash
cd cdk/devops
# Destroy the resources
cdk destroy
```

## Uninstall SonarQube

From the root folder of the tutorial repository, apply the following commands:

```bash
cd sonarqube
# Uninstall the SonarQube helm
helm uninstall -n sonarqube sonarqube-release
```

## Destroy your cluster

From the root folder of the tutorial repository, apply the following commands:

```bash
cd cdk/eks/addons
# Remove the addons
cdk deploy --context destroy=true
# Destroy the cluster
cd ..
cdk destroy
```

## Destroy your VPC

From the root folder of the tutorial repository, apply the following commands:

```bash
cd cdk/vpc
# Destroy the resources
cdk destroy
```
