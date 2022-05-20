
local wave = import 'wave.jsonnet';

{
  identities: {
    azure: wave.identities.Azure(name='foo-identity', resourceID='foo-resource-id', clientID='foo-client-id'),
  },

  objects: {
    serviceAccount: wave.objects.ServiceAccount(name='foo-service-account'),
    
    service: {
      empty: wave.objects.Service(name='foo-service'),
      ports: wave.objects.Service(name='foo-service') +
        wave.network.Port(name='same-port', port=8080) +
        wave.network.Port(name='different-port', port=8081, targetPort=8082),
    },

    headlessService: {
      empty: wave.objects.HeadlessService(name='foo-service'),
      ports: wave.objects.HeadlessService(name='foo-service') +
        wave.network.Port(name='same-port', port=8080) +
        wave.network.Port(name='different-port', port=8081, targetPort=8082),
    },

    externalService: {
      empty: wave.objects.ExternalService(name='foo-service', ip='1.2.3.4'),
      selector: wave.objects.ExternalService(name='foo-service', ip='1.2.3.4') +
        wave.features.CustomSelector(selector='foo-selector'),
    },

    deployment: {
      empty: wave.objects.Deployment(name='foo-deployment'),
      singleContainer: wave.objects.Deployment(name='foo-deployment') +
        wave.spec.DeploymentContainer(
          wave.objects.Container('foo-container', wave.env.Version('eu.gcr.io/foo')),
        ),
      multipleContainers: wave.objects.Deployment(name='foo-deployment') +
        wave.spec.DeploymentContainer(
          wave.objects.Container('foo-container', wave.env.Version('eu.gcr.io/foo')),
        ) +
        wave.spec.DeploymentContainer(
          wave.objects.Container('bar-container', 'eu.gcr.io/bar'),
        ),
      fullContainer: wave.objects.Deployment(name='foo-deployment') +
        wave.spec.DeploymentContainer(
          wave.objects.Container('foo-container', wave.env.Version('eu.gcr.io/foo')) +
          wave.network.ContainerPort(name='foo-port', port=8080) +
          wave.network.ContainerPort(name='bar-port', port=8081) +
          wave.features.DownwardAPI() +
          wave.features.healthchecks.HTTP() +
          wave.resources.Request(memory='256Mi'),
        ),
      azureBind: wave.objects.Deployment(name='foo-deployment') +
        wave.identities.AzureBind(name='foo-identity'),
      googleBind: wave.objects.Deployment(name='foo-deployment') +
        wave.spec.DeploymentContainer(
          wave.objects.Container('foo-container', 'eu.gcr.io/foo') +
          wave.identities.GoogleBind()
        ) +
        wave.identities.Google(name='foo-deployment'),
      env: wave.objects.Deployment(name='foo-deployment') +
        wave.spec.DeploymentContainer(
          wave.objects.Container('foo-container', wave.env.Version('eu.gcr.io/foo')) +
          wave.env.Custom(name='FOO_ENV', value='foo-value')
        ),
    },

    statefulset: {
      empty: wave.objects.StatefulSet(name='foo-statefulset'),
      singleContainer: wave.objects.StatefulSet(name='foo-statefulset') +
        wave.spec.StatefulSetContainer(
          wave.objects.Container('foo-container', wave.env.Version('eu.gcr.io/foo')),
        ),
      volumes: wave.objects.StatefulSet(name='foo-statefulset') +
        wave.spec.StatefulSetContainer(
          wave.objects.Container('foo-container', wave.env.Version('eu.gcr.io/foo')) +
            wave.volumes.Mount(name='foo-volume', path='/etc/path'),
        ) +
        wave.volumes.Google(name='foo-volume', disk='foo-disk'),
      env: wave.objects.StatefulSet(name='foo-statefulset') +
        wave.spec.StatefulSetContainer(
          wave.objects.Container('foo-container', wave.env.Version('eu.gcr.io/foo')) +
          wave.env.Custom(name='FOO_ENV', value='foo-value'),
        ),
    },

    cronjob: {
      empty: wave.objects.CronJob(name='foo-cronjob', schedule='* * * * *'),
      singleContainer: wave.objects.CronJob(name='foo-cronjob', schedule='* * * * *') +
        wave.spec.CronJobContainer(
          wave.objects.Container('foo-container', wave.env.Version('eu.gcr.io/foo')),
        ),
    },
  },

  features: {
    Sentry: wave.objects.Deployment(name='foo-sentry') +
      wave.spec.DeploymentContainer(
        wave.objects.Container('foo-container', 'eu.gcr.io/foo') +
        wave.features.Sentry('foo-sentry-project')
      )
  },
}
