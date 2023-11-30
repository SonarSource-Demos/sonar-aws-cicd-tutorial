
# Implementing Sonar Clean Code on AWS

## Introduction

This content was initially assembled to support the [Sonar/AWS joint workshop Clean Code in CI/CD Pipelines with SonarQube, AWS CodePipeline, and Amazon EKS](https://aws-experience.com/emea/smb/xe/4b788/workshop-clean-code-in-cicd-pipelines-with-sonarqube-aws-codepipeline-and-amazon-eks-zurich) held in August, 2023 in Zurich.
It has been enriched to support individuals wanting to explore the topic on their own, as a tutorial.

## Content structure

The following folders represent the main chapters of the tutorial

- `1. Introduction` describes the overal scenario
- `2. Clean Code` introduces the topic of Clean Code describes how to deploy SonarQube on EKS
- `3. CI/CD-pipeline` instructs on how to setup the CI/CD pipeline in AWS for a sample Java project
- `4. Improve Code` instructs people how to analyze the application code with Sonar, deploy the app, then improve and re-deploy the app
- `5. Clean up` provides basic instructions on how to clean-up any ressources that were created on AWS for the tutorial

On each page of this tutorial, at the bottom, you'll find a ’next’ and a ’previous’ link that will allow you to progress through the content.

### What to expect from this tutorial?

Imagine you are a senior developer joining a new team. The team, working in DevOps mode, is struggling with frequent production outages caused by bugs in their application code. Also, shipping new code to their Kubernetes cluster on Amazon EKS is a manual process.
You will familiarize yourself with the existing infrastructure on AWS. Then you will deploy SonarQube and build a CI/CD process using AWS CodeBuild and AWS CodePipeline to improve code quality and ease of deployment.

### At the end of this tutorial, you will be able to:

- Understand the concept of Clean Code
- Deploy SonarQube on Amazon EKS
- Build a CI/CD process on AWS integrating Sonar code analysis
- Use AWS CodeBuild to build container images
- Use AWS CodePipeline to build a serverless CI/CD pipeline

### What you should NOT expect from this tutorial?

- a ready-to-go production setup. Some shortcuts were taken to reduce the amount of time needed to build the complete stack, some of them should not be used for your prodution. Comments about it will be provided in the different chapters.
- a fit-for-all DevOps stacks. The proposed pipeline and deployment helpers may not entirely fit with your existing projects and AWS environment.
- detailed troubleshooting instructions for any problem you may encounter with the proposed DevOps stack
- Support. The resources hosted here are provided 'as is', feel free to report any problem or suggest improvements as an Issue on the project. This tutorial is not covered by the Sonar or AWS standard support channels, we'll do our best, without any SLA.

### How much time does it take to complete the tutorial?

Approximately 4 hours

### What is the required level?

You need to have a basic understanding of an object-oriented programming language. It will also help if you have a basic understanding of Kubernetes and AWS.

### What tools do I need?

You can execute most of the steps in this tutorial with a browser and access to an AWS account only. If you want to execute [SonarLint](/cleancode/install-sonarlint) you need to clone a code repository to your local machine (using Git) and you need to use a local IDE (either [Visual Studio Code](https://code.visualstudio.com/) or [IntelliJ](https://www.jetbrains.com/idea/))

### Will this cost me anything?

If you execute the steps in this tutorial in your own AWS account, this tutorial will incur some costs. Make sure you clean up your resources at the end to avoid unnecessary charges.

### Which AWS Regions can I use?

Feel free to use the region of your choice to run this tutorial!

-----------------
[Previous](./README.md) | [Next](./1-Introcution/README.md)