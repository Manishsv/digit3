# Configuring DIGIT Service

To get started with our module, we need to interact with DIGIT platform services to create and configure the account, users, roles and all the DIGIT service templates needed for this PGR module. Luckily this task is made super simple with just a few lines of code using DIGIT CLI!

DIGIT CLI is a comprehensive command-line interface for interacting with DIGIT platform services. This CLI tool provides commands for account management, user creation, role assignment, workflow management, template creation, MDMS operations, and more. To learn more about DIGIT CLI, please click [here](https://github.com/digitnxt/digit3/tree/654c85117283c46e9112f92616221d67ede66f69/tools/digit-cli)

Follow along with [this video](https://github.com/digitnxt/digit3/blob/1eb4ef483a7e95577c32d0bbf27df158d50fe9e5/docs/tutorials/backend/Configure%20w%20DIGIT%20CLI.mp4):


Follow the following steps to configure the PGR module:

0. Install DIGIT CLI
Please refer [here:](https://github.com/digitnxt/digit3/tree/654c85117283c46e9112f92616221d67ede66f69/tools/digit-cli) to install DIGIT CLI based on your OS.

After installation, try the command:
```bash
digit --help
```

If you see this:


<img width="778" height="346" alt="Screenshot 2025-11-21 at 12 26 23 AM" src="https://github.com/user-attachments/assets/c792ef9b-4674-4bc0-9ab5-29142bfa4084" />

Your DIGIT CLI is ready to roll!

## 1. CREATE AN ACCOUNT
In the context of software architecture, an "account" (previously refered to a tenant) typically refers to an individual or organization that uses a shared software application or service. Each tenant operates within its isolated portion of the application's resources, such as data, configuration, and user interface. To learn more click [here](https://docs.digit.org/faqs/the-concept-of-tenant-in-digit#what-is-a-tenant)

Let us create an account <b>Amaravati</b> for this tutorial, with email: <b>test@example.com.</b>.

```bash
digit create-account --name Amaravati --email test@example.com --server https://digit-lts.digit.org
```            
Note: This may take some time.

## 2. CONFIGURE THE ACCOUNT
Customize the account’s experience and settings after sign-up, by setting up the account with relevant parameters.

```bash
digit config set --server https://digit-lts.digit.org --account AMARAVATI --client-id auth-server --client-secret changeme --username test@example.com --password default
```  

## 3. IDGEN CONFIGURATION: Configure the template we require for ID generation in the PGR module:

The template-code being used is 'pgr'.
Note: Keep this in mind, it will be used while generating the springboot application soon!

```bash
digit create-idgen-template --template-code pgr --template "{ORG}-{DATE:yyyyMMdd}-{SEQ}-{RAND}" --scope daily --start 1 --padding-length 4 --padding-char "0" --random-length 2 --random-charset "A-Z0-9"
```
Configure the registryId template we require for ID generation in the PGR module:

```bash
digit create-idgen-template --template-code registryId --template "{ORG}-{DATE:yyyyMMdd}-{SEQ}-{RAND}" --scope daily --start 1 --padding-length 4 --padding-char "0" --random-length 2 --random-charset "A-Z0-9"
```
## 4. WORKFLOW CONFIGURATION: Configure the process, states and actions we require for workflow transitions in the PGR module:

Here we are using the default configuration which has all the process, states and actions we require. 

```bash
digit create-workflow  --code PGR --default
```

## 5. NOTIFICATION CONFIGURATION: Configure the template we require for sending notifications in the PGR module:

The template-code being used is 'my-template'.

```bash
digit create-notification-template --template-id "my-template" --version "1.0.0" --type "EMAIL" --subject "Test Subject" --content "Test Content"
```
## 6. CREATING AN USER IN ACCOUNT

Let us create a user called John Doe with password, mypassword and email, john@example.com.

```bash
digit create-user --username johndoe --password mypassword --email john@example.com
```

and make sure the user is created correctly and exists!

```bash
digit search-user --username johndoe
```

## 7. CREATE ROLE

Here we are creating the role of CSR, i.e., Citizen Service Representative.

```bash
digit create-role --role-name CSR --description "Administrator role"
```

## 8. ASSIGN ROLE TO USER

First we must make Johndoe a SUPERUSER.

```bash
digit assign-role --username johndoe --role-name SUPERUSER
```

Let us assign our created user Johndoe, the role of CSR.

```bash
digit assign-role --username johndoe --role-name CSR
```

## 9. BOUNDARY CONFIGURATION: Configure the boundary hierarchy, boundaries and boundary relationships require for the PGR module:

Here we are using the default configuration which has all the boundary hierarchy, boundaries and boundary relationships we require. 

```bash
digit create-boundaries --default
```

## 10. REGISTRY SCHEMA: 

Here we are using the file 'pgr2-registry-schema.yaml' which has all the process, states and actions we require. You can download this file [here](https://github.com/digitnxt/digit3/blob/2be321440a9dfb65d1fb344035c1e166699eeaf1/docs/tutorials/backend/pgr2-registry-schema.yaml) and copy the path.

```bash
digit create-registry-schema --file /Users/srujana/Downloads/pgr2-registry-schema.yaml
```

That's it! We are ready for developing our module on DIGIT 3.0.
