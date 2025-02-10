# Harbor Cleaner

Harbor's built-in GC feature does not support deleting images based on specific tags. This tool was created to clean up unnecessary tags, such as those containing "rc", "hotfix", and others.

## Usage

1. **Clone the repository**:

```bash
git pull https://github.com/warnerchen/harbor-cleaner.git
```

2. **Set the required environment variables**:

```bash
export HARBOR_REGISTRY=https://harbor.warnerchen.com
export HARBOR_USERNAME=harbor-cleaner
export HARBOR_PASSWORD=xxxxxx
export HARBOR_PROJECTS=library,rancher
```

- `HARBOR_REGISTRY`: Your Harbor registry URL.
- `HARBOR_USERNAME`: Username for authentication.
- `HARBOR_PASSWORD`: Password for authentication.
- `HARBOR_PROJECTS`: Comma-separated list of Harbor projects to clean.
- `HARBOR_TAGS`: This uses a fuzzy match, so any tag containing the specified strings will be deleted (e.g., "rc" will match "rc1-xxx", "v2.10.1-rc2", etc.)
  
3. **Navigate to the cloned repository and run the tool**:

```bash
cd harbor-cleaner
go build -o harbor-cleaner main.go
./harbor-cleaner
```

## Deploy to Kubernetes

1. **Build the image**:

```bash
docker build -t <registry>/library/harbor-cleaner:latest . -f deploy/Dockerfile
docker push <registry>/library/harbor-cleaner:latest
```

2. **Deploy**:

```bash
# Replace the values in deploy/configmap.yaml with your own values.
kubectl apply -f deploy/configmap.yaml

# Replace the values in deploy/secret.yaml with your own values.
kubectl apply -f deploy/secret.yaml

# Replace the image in deploy/cronjob.yaml with your image name.
kubectl apply -f deploy/cronjob.yaml
```