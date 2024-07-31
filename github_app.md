
# GitHub App configuration

- Create a GitHub App
  - Go to <https://github.com/settings/apps>
  - New GitHub App
  - Fill in the details:
    - `Homepage URL`: URL of the repo
    - Create private key and download it, later will be stored in AWS secret manager
    - Install the app to the repo
      - <https://github.com/settings/apps/APP_NAME/installations>
      - `Only selected repositories`
        - Select repositories where the app will be installed and be used for `pull-bot`
    - Go to <https://github.com/settings/apps/APP_NAME/permissions> and select
      - `Permissions`
        - Read access to `commit statuses` and `metadata`
        - Read and write access to `code`, `issues`, `pull requests`, and `workflows`

      __NOTE__ Check email and approve the permissions once you saved it
    - Copy the `App ID` and `App Name` to the `.env` file

## Local testing of GitHub App and Image

```bash
docker run -it \
  --env-file .env \
  -v PRIVATE_KEY.pem:/app/PRIVATE_KEY.pem \
  -p 3000:3000 \
  ghcr.io/g-research/pull:latest
```

Example of `.env` file

```bash
APP_ID=XXX
APP_NAME=XXX
PULL_INTERVAL=30    # default 3600 seconds
JOB_TIMEOUT=60      # default 60 seconds
DEFAULT_MERGE_METHOD=hardreset
```
