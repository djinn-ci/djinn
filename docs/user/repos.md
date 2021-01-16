[Prev](/user/namespaces) - [Next](/user/cron)

# Repos

If you're connected to either GitHub or GitLab you will have the ability to
enable webhooks on the repositories you have with these providers. You can
connect to these providers by either logging in with them, or by managing
these connections from your account [settings]({{index .Vars "host"}}/settings).

Once connected you can visit the [repos]({{index .Vars "host"}}/repos) that you
own, and enable them to integrate with Djinn CI. This will add a webhook to the
enabled repository that will submit a new build to Djinn CI on a new push or
pull request.

For this webhook to work you will need to add a `.djinn.yml` file to the root
of the repository, or add a `.djinn` directory to the root of the repository
containing the build manifests you wish to submit. Each YAML file in this
directory must be suffixed with the `.yml` extension.
