# EC-CSI-Driver

## Описание:

Этот Helm chart разворачивает CSI (Container Storage Interface) плагин для взаимодействия с облачным хранилищем. Он включает в себя компоненты для контроллера и ноды, а также конфигурацию StorageClass и VolumeSnapshotClass для управления томами и снапшотами.

## Предварительные требования:

- Kubernetes кластер. Протестирован с версией v1.31.0
- Helm

## Описание параметров конфигурации:

- `global.namespace`: Пространство имен, в котором будут развернуты компоненты CSI.
- `global.clusterID`: Уникальный идентификатор кластера.
- `global.imagePullSecrets`: Секрет Kubernetes, содержащий учетные данные для доступа к реестру контейнеров. 
- `global.dockerconfigjson`: JSON-конфигурация для аутентификации в реестре контейнеров.
- `images`: Раздел, содержащий информацию о репозиториях и тегах образов контейнеров для различных компонентов CSI.
- `storageClasses`: Раздел, определяющий конфигурацию StorageClass для различных типов хранилищ.
  * `name`: Имя StorageClass.
  * `type`: Тип хранилища.
  * `setAsDefault`: Указывает, является ли StorageClass StorageClass по умолчанию.
  * `allowVolumeExpansions`: Разрешает ли StorageClass расширение томов.
  * `snapshotRestoreEnabled`: Разрешает ли StorageClass восстановление из снапшотов.
- `snapshotClass`: Конфигурация для VolumeSnapshotClass.
  * `name`: Имя VolumeSnapshotClass.
  * `deletionPolicy`: Политика удаления снапшотов.
- `groupSnapshotClass`: Конфигурация для VolumeGroupSnapshotClass для групповых снапшотов.
  * `name`: Имя VolumeGroupSnapshotClass.
  * `deletionPolicy`: Политика удаления снапшотов.