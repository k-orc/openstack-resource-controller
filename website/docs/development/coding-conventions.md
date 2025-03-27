# Coding conventions
## Logging

We have [4 logging levels](https://github.com/k-orc/openstack-resource-controller/blob/main/internal/logging/levels.go): `Status`, `Info`, `Verbose` and `Debug`.
The first three are meant for administrators and the last one for developers.

### Status Level

`Status` logs are always shown. Status logs should be reserved for operational logs about the service itself, e.g.:
- Startup and shutdown messages
- Runtime conditions which may indicate something about the state of the service, e.g. inability to reach kube-apiserver.

### Info Level

`Info` is the default log level for most deployments. It should log the principal actions
of the service, i.e. resource creation and deletion, and 'reconcile complete' (i.e. Progressing=False) messages for success and failure. It should not include actions which happen on every reconcile.  
Example: ```OpenStack resource created```

### Verbose Level

`Verbose` logs provide additional context for an administrator trying to understand why an action
be occurring or not occurring. It should produce logs on every reconcile attempt.  
Example: ```web-download is not supported because...```

### Debug Level

`Debug` logs are very verbose. They should include things that should help with debugging/development.  
Example: ```Got resource```
