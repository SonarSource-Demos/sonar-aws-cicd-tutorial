# Development Workflow

This last phase of our tutorial will highlight what's happening in a regular Development workflow, involving a development branch and a pull request.

* A new branch is created to add some use case. That branch was already created for you, it's named ```new-service```
* Create a Pull-Request to merge the ```new-service``` commits into ```main```
  * if there is any PR conflict, address them by keeping the destination branch content
* CodeBuild is running on the PR code and will validate your changes
  * a second PR analysis is immediatley launched after the first if you fix any conflict with a new commit.
  * your CodeBuild PR run(s) will fail because the Quality Gate does not pass, as described with [Failing a pipeline job when the quality gate fails | SonarQube Docs](https://docs.sonarsource.com/sonarqube/latest/analyzing-source-code/ci-integration/overview/#quality-gate-fails)

![Quality Gate](/assets/4.DevWorkFlow/qualitygate.png)

* You may now explore the issues that are breaking the Quality Gate, and fix them on your new-service branch.
* Don't hesitate to explore how your new-service branch is vulnerable to SQL injection locally
  * To display arbitrary values on the page: <http://127.0.0.1:8080/person/address/?name='+UNION+SELECT+'something-is-wrong>
  * To extract the data from all users of the table PEOPLE: <http://127.0.0.1:8080/person/address/?name='+UNION+SELECT+ARRAY_AGG(CONCAT(name,address))+FROM+PEOPLE-->
  * To insert new data in the table people PEOPLE, note that this request will return an error because the initial SELECT fails, and then we do the INSERT: <http://127.0.0.1:8080/person/address/?name=';INSERT+INTO+PEOPLE(NAME,ADDRESS)+VALUES('not-expected','nowhere')-->
  * To truncate the table PEOPLE: <http://127.0.0.1:8080/person/address/?name=';DELETE+FROM+PEOPLE-->
* Use SonarLint and the Connected Mode to help you fix the new issues
  * If you're stuggling to add Unit Tests (which is not the point of this tutorial), you may reduce the [scope of analysis for test coverage](https://docs.sonarsource.com/sonarqube/latest/project-administration/analysis-scope/#code-coverage-exclusion)
* Your PR will get analyzed again after you push you new commit(s) to the CodeCommit repository
* Once thall issues are adressed, SonarQube Quality Gate will be green and your CodeBuild run will pass

![Passed QualityGate](/assets/4.DevWorkflow/passedQualityGate.png)

Your PR is 'Clean Code-Ready', you may now merge it and get the new functionalities deployed.

## The End

You've reached the end of this tutorial, congrats!

---
[Previous](../3.DevOps/README.md)|[Clean Up](../5-CleanUp/README.md)
