# what to do in rewrite
- move the configuration of services to the service itself, as in the repo
- better splitting of components, main problem was self-updating the main daemon
- better separation of concerns, the main daemon should not be responsible for updating itself
- improve logging, make it much more verbose