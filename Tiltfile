# Tiltfile
# 1. SETUP
# Replace with your actual registry (REQUIRED for remote clusters)
default_registry('harbor.mark3748.com/helpdesk-go') 

# Define the path to your local overrides
local_values_file = 'helm/local-values.yaml'
values_files = ['helm/dev-values.yaml']

# If the local file exists, add it to the list
if os.path.exists(local_values_file):
    values_files.append(local_values_file)

# Force the namespace for all resources
k8s_namespace('helpdesk-dev')
allow_k8s_contexts('default')

# 2. BUILD
docker_build('helpdesk-api', '.', dockerfile='Dockerfile.api')
docker_build('helpdesk-worker', '.', dockerfile='Dockerfile.worker')
docker_build('helpdesk-internal-frontend', '.', dockerfile='Dockerfile.frontend-internal')
docker_build('helpdesk-requester-frontend', '.', dockerfile='Dockerfile.frontend-requester')

# 3. DEPLOY DEPENDENCIES
# Assumes k8s/kustomization.yaml exists as discussed
k8s_yaml(kustomize('k8s'))

# 4. DEPLOY HELM CHART
k8s_yaml(helm(
    'helm/helpdesk',
    values=values_files,
    namespace='helpdesk-dev',
    set=[
        # These match the docker_build names above
        'image.repository=helpdesk-api',
        'workerImage.repository=helpdesk-worker',
        'frontendInternal.image.repository=helpdesk-internal-frontend',
        'frontendRequester.image.repository=helpdesk-requester-frontend'
    ]
))

# 5. ORGANIZE UI
# Group the dependencies
k8s_resource('postgres', labels=['dependencies'])
k8s_resource('redis', labels=['dependencies'])

# Group the Helm resources into a single "App" label
# Note: We use the names Tilt explicitly told you it found in the error message
k8s_resource('chart-helpdesk-api', new_name='api', labels=['app'])
k8s_resource('chart-helpdesk-worker', new_name='worker', labels=['app'])
k8s_resource('chart-helpdesk-frontend-internal', new_name='internal-web', labels=['app'])
k8s_resource('chart-helpdesk-frontend-requester', new_name='requester-web', labels=['app'])
