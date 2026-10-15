This is Hephaestus. A git forge and runtime for autonomous programs.

# Entities

## Git repositories

Git repositories : named and slugged. They're stored on the host disk.

## OCI Images

OCI Images are OCI images. named and slugged.

## VMs

A VM is an instance of a VM.

They are based on an OCI Image.
They are provisioning, active, stopped or deleted.

## Volumes

A volume is a named disk space that can be mounted in a VM.

## Projects

Projects have names and slugs.

They can have Git repositories, secrets, volumes, agents

## Secrets and Environment variables

Secrets and envvariables ar key/value pairs.

{find good pactices for secrets}

## Users

Users can be human, organisations, agents

Users can have projects.
Users can have secrets.


### Humans

Humans have logins and passwords, or other ways of logging in. They can use the UI app.

### Organisations
### Agents

How do agents authentify with Heph ?
Agents have an identity, an Agent Definition

## OCI Image definition

An agent can expose a way to build custom images. The repo itself should be available during the build process.

## Agent Definitions

A git Repo can expose agent definitions.
An agent definitions includes:

 - An image reference (or a Dockerfile as an option ?)
 - VM requirements ?
 - A build process (typically you will build and copy your git repo)

 


 
# Permissions

There are permissions. All API actions are subject to permissions.

Permissions should be a generic plug-in

# Platform

- Git server
- API Server
- Permission engine
- HTTP proxy
