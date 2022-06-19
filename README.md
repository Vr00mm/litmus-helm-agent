# litmus-helm-agent
This is unofficial helm chart for litmus helm agent for litmus 2.8 (release -2)
CLUSTER SCOPE ONLY

## Install local agent (same cluter):

```
git clone https://github.com/Vr00mm/litmus-helm-agent
cd litmus-helm-agent
helm upgrade --install \
  litmus-agent helm/litmus-agent \
  --namespace litmus --create-namespace \
  --set "AGENT_NAME=helm-agent" \
  --set "AGENT_DESCRIPTION=My first agent deployed with helm !" \
  --set "LITMUS_URL=http://litmusportal-frontend-service.litmus.svc.cluster.local:9091" \
  --set "LITMUS_BACKEND_URL=http://litmusportal-server-service.litmus.svc.cluster.local:9002" \
  --set "LITMUS_USERNAME=admin" \
  --set "LITMUS_PASSWORD=litmus" \
  --set "LITMUS_PROJECT_ID=69365cb3-0211-4262-8820-78056c8adb4c"
```

## Install remote agent:

```
git clone https://github.com/Vr00mm/litmus-helm-agent 
cd litmus-helm-agent
helm upgrade --install \
  litmus-agent helm/litmus-agent \
  --namespace litmus --create-namespace \
  --set "AGENT_NAME=helm-agent" \
  --set "AGENT_DESCRIPTION=My first agent deployed with helm !" \
  --set "LITMUS_URL=http://my-domain.com" \                       
  --set "LITMUS_USERNAME=admin" \
  --set "LITMUS_PASSWORD=litmus" \
  --set "LITMUS_PROJECT_ID=69365cb3-0211-4262-8820-78056c8adb4c"
```

## No trust docker image ?

```
git clone https://github.com/Vr00mm/litmus-helm-agent 
cd litmus-helm-agent

docker build . -f docker/Dockerfile -t your-docker-repo/your-image-name:your-tag
docker push your-docker-repo/your-image-name:your-tag

helm upgrade --install \
  litmus-agent helm/litmus-agent \
  --namespace litmus --create-namespace \
  --set "AGENT_NAME=helm-agent" \
  --set "AGENT_DESCRIPTION=My first agent deployed with helm !" \
  --set "LITMUS_URL=http://my-domain.com" \
  --set "LITMUS_USERNAME=admin" \
  --set "LITMUS_PASSWORD=litmus" \
  --set "LITMUS_PROJECT_ID=69365cb3-0211-4262-8820-78056c8adb4c" \
  --set "image.repository=your-docker-repo/your-image-name \
  --set "image.tag=your-tag"
```

## Any bug ?

Do not hesitate report bug or issue !!
