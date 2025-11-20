# Backend Developer Guide

This guide provides detailed steps for developers to create a new microservice on top of DIGIT 3.0. At the end of this guide, you will be able to run the [PGR module](https://github.com/digitnxt/digit3/tree/pgr-demo/pgrown3.0_copy/src/main) provided and test it out locally.

**Steps to create a microservice:**

* Set up your development environment
* Develop the registries, services, and APIs for a voter registration module that were described in the [Design Guide](https://docs.digit.org/platform/guides/design-guide)
* Integrate with an existing DIGIT environment and re-use a lot of the common services using [DIGIT Client Library](https://github.com/digitnxt/digit3/tree/client-library-java-7/code/digit-client/src/main/java/com/digit/services)
* Test the new module
* Build the new service in the DIGIT environment

The guide is divided into multiple sections for ease of use. Click on the section cards below to follow the development steps.

<table data-view="cards"><thead><tr><th></th><th></th><th></th></tr></thead><tbody><tr><td><a href="backend-developer-guide/section-0-prep"><mark style="color:blue;"><strong>Section 0: System Setup</strong></mark></a></td><td>Learn all about the development pre-requisites, design inputs, and environment setup</td><td></td></tr><tr><td><a href="backend-developer-guide/section-1-create-project"><mark style="color:blue;"><strong>Section 1: Configuring DIGIT Service</strong></mark></a></td><td>Configure the account, users, roles and all the DIGIT service templates needed for this PGR module using DIGIT CLI.</td><td></td></tr><tr><td><a href="backend-developer-guide/section-2-integrate-persister-and-kafka"><mark style="color:blue;"><strong>Section 2: Generate Project</strong></mark></a></td><td>Generate most of your spring boot project in (almost) 1 click!</td><td></td></tr><tr><td><a href="backend-developer-guide/section-3-integrate-microservices"><mark style="color:blue;"><strong>Section 3: Creating Domain layer</strong></mark></a></td><td>Steps on converting your generated DTOs to Entities.</td><td></td></tr><tr><td><a href="backend-developer-guide/section-4-integrate-billing-and-payment"><mark style="color:blue;"><strong>Section 4: Tying up the loose ends of your module</strong></mark></a></td><td>Finishing touches to controller layer and cleaning up the project.</td><td></td></tr><tr><td><a href="backend-developer-guide/section-5-other-advanced-integrations"><mark style="color:blue;"><strong>Section 5: Integrating your module with DIGIT Services</strong></mark></a></td><td>Learn how to integrate services to the built module using DIGIT Client Library</td><td></td></tr><tr><td><a href="backend-developer-guide/section-6-run-final-application"><mark style="color:blue;"><strong>Section 6: Run Application</strong></mark></a></td><td>Test run the built application in the local environment</td><td></td></tr><tr><td><a href="backend-developer-guide/section-7-build-and-deploy-instructions"><mark style="color:blue;"><strong>Section 7: Build &#x26; Deploy Instructions</strong></mark></a></td><td>Deploy and run the modules</td><td></td></tr></tbody></table>

Access the sample PGR module [here]([https://github.com/digitnxt/digit3/tree/pgr-demo/pgrown3.0_copy/src/main]). Download and run this in the local environment.
