# PAC - Draft Content API

Draft content API is a microservice that provides access to draft content stored in PAC.

## Code

draft-content-api

## Primary URL

<https://upp-prod-delivery-glb.upp.ft.com/__draft-content-api/>

## Service Tier

Bronze

## Lifecycle Stage

Production

## Delivered By

content

## Supported By

content

## Known About By

- dimitar.terziev
- hristo.georgiev
- elitsa.pavlova
- elina.kaneva
- kalin.arsov
- ivan.nikolov
- miroslav.gatsanoga
- mihail.mihaylov
- tsvetan.dimitrov
- georgi.ivanov
- robert.marinov

## Host Platform

AWS

## Architecture

Provides endpoints for saving draft content. Draft content is written in native CMS format to /drafts/nativecontent/{uuid}.
Draft content may only be read in UPP format from /drafts/content/{uuid} if the service is called internally
(because that URL is used externally by Draft Content Public Read). When used for reading the Draft Content API calls
the necessary mapper/validator to perform transformation on-demand. If draft content is not available in PAC for a document,
the Draft Content API may retrieve the most recently published version from UPP.

## Contains Personal Data

No

## Contains Sensitive Data

No

## Dependencies

- generic-rw-aurora
- pac-methode-article-mapper
- pac-upp-content-validator

## Failover Architecture Type

ActiveActive

## Failover Process Type

FullyAutomated

## Failback Process Type

FullyAutomated

## Failover Details

The service is PAC cluster.
The failover guide for the cluster is located here:
<https://github.com/Financial-Times/upp-docs/tree/master/failover-guides/pac-cluster>

## Data Recovery Process Type

NotApplicable

## Data Recovery Details

The service does not store data, so it does not require any data recovery steps.

## Release Process Type

PartiallyAutomated

## Rollback Process Type

Manual

## Release Details

Manual failover is needed when a new version of
the service is deployed to production.
Otherwise, an automated failover is going to take place when releasing.
For more details about the failover process please see: <https://github.com/Financial-Times/upp-docs/tree/master/failover-guides/pac-cluster>

## Key Management Process Type

Manual

## Key Management Details

To access the service clients need to provide basic auth credentials.
To rotate credentials you need to login to a particular cluster and update varnish-auth secrets.

## Monitoring

Service in UPP K8S PAC clusters:

- PAC-Prod-EU health: <https://pac-prod-eu.ft.com/__health/__pods-health?service-name=draft-content-api>
- PAC-Prod-US health: <https://pac-prod-us.ft.com/__health/__pods-health?service-name=draft-content-api>

## First Line Troubleshooting

<https://github.com/Financial-Times/upp-docs/tree/master/guides/ops/first-line-troubleshooting>

## Second Line Troubleshooting

Please refer to the GitHub repository README for troubleshooting information.
