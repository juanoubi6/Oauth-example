# OAUTH EXAMPLE

Simple google oauth example to check if user email has been given a custom created role in our google project.

1) Create google project
2) Create oauth key in the Apis & Services -> Credentials -> Create credential (oauth client ID). Assign it the callback url of your server
3) Create a new role from IAM & admin -> Roles. Assign "iam.roles.get" permissions (still have to check this)
4) Enable "Cloud Resource Manager API" for your project in APIs & services -> Dashboard
5) Create a new service account (download it's JSON credentials) and assign it the "Owner" role (or a role that has permissions for the API you need to request)
