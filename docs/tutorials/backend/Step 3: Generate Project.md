# Generate Project Using API Specs - In 1 command! (Almost)

This page provides detailed steps for generating projects using the given API specifications via OpenAPI generator. 

## Steps

### 0. Clone this repo.
```bash
git clone --branch openapigenerator-custom-file https://github.com/digitnxt/digit3.git
```

### 1. Navigate to the folder 'gen-DIGIT-stub'
```bash
cd digit3
cd gen-DIGIT-stub
```

### 2. Use this command to generate most of your project stub.

```bash
openapi-generator generate \
  -g spring \
  -i pgr.yaml \
  -o generated-pgr \
  -t ./final-custom-templates \
  -c config.yaml \
  --api-package com.example.pgrown30.web.controllers \
  --model-package com.example.pgrown30.web.models \
  --invoker-package com.example.pgrown30.web \
  --additional-properties=\
packageName=com.example.pgrown30,\
servicePackage=com.example.pgrown30.web.service,\
useSpringBoot3=true,\
groupId=com.example,\
artifactId=pgrown30,\
artifactVersion=0.0.1-SNAPSHOT,\
name=pgrown30,\
description="PGR Service generated from YAML",\
dbName=pgrown,\
dbSchema=public,\
dbUser=postgres,\
dbPassword=1234,\
serverPort=8083,\
idgenTemplateCode=pgr,\
WorkflowProcessCode=PGR67,\
RegistrySchemaCode=pgr2 \
  --global-property=apis,models,supportingFiles
````
Your pom.xml, application.properties, models, controller file and config files are all generated. 

NOTE1: Change the flags idgenTemplateCode, WorkflowProcessCode, RegistrySchemaCode to what was configured in CLI accordingly.
NOTE2: Make sure the base URLS are as below:

```bash
digit.services.boundary.base-url=http://localhost:8095
digit.services.workflow.base-url=http://localhost:8095
digit.services.idgen.base-url=http://localhost:8095
digit.services.notification.base-url=http://localhost:8095
digit.services.filestore.base-url=http://localhost:8095
digit.services.registry.base-url=http://localhost:8095
```

### 2. Import in IDE.

Now you have all the generated files you need! Open the generated-pgr folder in the IDE of your choice. Make sure it is using this folder structure:


<img width="233" height="383" alt="Screenshot 2025-12-22 at 12 33 28 PM" src="https://github.com/user-attachments/assets/4423ae9c-f41f-459f-a74b-22f6c56e6414" />


Most of your PGR module has been generated for you! 
