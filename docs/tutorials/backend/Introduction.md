# Backend Developer Guide

This guide provides detailed steps for developers to create a new microservice on top of DIGIT 3.0. At the end of this guide, you will be able to run the [PGR module](https://github.com/digitnxt/digit3/tree/11b02a69a501282acceea9c93172ff584025c12a/pgrown3.0_copy) provided and test it out locally.

**Steps to create a microservice:**

* Set up your development environment
* Configure DIGIT services using [DIGIT CLI](https://github.com/digitnxt/digit3/tree/de5d714c59919e61e68a83072badde6f4b2a0ae4/tools/digit-cli)
* Develop the registries, services, and APIs for the module that were described in the [Design Guide](https://docs.digit.org/platform/guides/design-guide)
* Integrate with an existing DIGIT environment and re-use a lot of the common services using [DIGIT Client Library](https://github.com/digitnxt/digit3/tree/de5d714c59919e61e68a83072badde6f4b2a0ae4/src/libraries/digit-client)
* Test the new module

The guide is divided into multiple sections for ease of use. Click on the section cards below to follow the development steps.

Please note that step 2 can be done via LLM are any other way as per your convenience. 

<table data-view="cards">
  <thead>
    <tr>
      <th>Section</th>
      <th>Description</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>Section 0: System Setup</strong></td>
      <td>Learn all about the development pre-requisites, design inputs, and environment setup</td>
    </tr>
    <tr>
      <td><strong>Section 1: Configuring DIGIT Services</strong></td>
      <td>Configure the account, users, roles and all the DIGIT service templates needed for this PGR module using DIGIT CLI.</td>
    </tr>
    <tr>
      <td><strong>Section 2: Generate Project</strong></td>
      <td>Generate most of your spring boot project in (almost) 1 click!</td>
    </tr>
    <tr>
      <td><strong>Section 3: Fill the logic in Controller Layer</strong></td>
      <td>Converting the generated controller stub to a complete controller layer</td>
    </tr>
    <tr>
      <td><strong>Section 4: Creating Service Layer</strong></td>
      <td>Creating the service implementation logic.</td>
    </tr>
    <tr>
      <td><strong>Section 5: Creating Validation Layer</strong></td>
      <td>This layer performs checks that depend on external platform data, such as boundaries or file references.</td>
    </tr>
    <tr>
      <td><strong>Section 6: Creating Client Layer</strong></td>
      <td>The client layer turns platform integrations into simple method calls.</td>
    </tr>
    <tr>
      <td><strong>Section 7: Run and Test your module</strong></td>
      <td>Test run the built application in the local environment</td>
    </tr>
  </tbody>
</table>


Access the sample PGR module [here](https://github.com/digitnxt/digit3/tree/11b02a69a501282acceea9c93172ff584025c12a/pgrown3.0_copy). Download and run this in the local environment.
