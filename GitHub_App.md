
# GitHub App configuration

- Create a GitHub App
  - Go to <https://github.com/settings/apps>
  - New GitHub App
  - Fill in the details:
    - `Homepage URL`: URL of the repo
    - `Webhook`: set to Active
      - `Webhook URL`: Create new channel <https://smee.io/> URL used for local testing
      - `Webhook Secret`: Secret used in `.env` file
    - Create private key and download it, later will be stored in AWS secret manager
    - Install the app to the repo
      - <https://github.com/settings/apps/APP_NAME/installations>
      - `Only selected repositories`
        - Select repositories where the app will be installed and be used for `pull-bot`
    - Go to <https://github.com/settings/apps/APP_NAME/permissions> and select
      - `Permissions`
        - XXX
      - `Events`
        - XXX
      __NOTE__ Check email and approve the permissions once you saved it
    - Copy the `App ID` and `App Name` and `Webhook secret` to the `.env` file
