# sshroute

## GitHub Actions

Always pin actions to a full commit SHA, never use a tag or branch reference alone.
Include the version as a comment for readability:

```yaml
uses: actions/checkout@<full-sha>  # v6.0.2
```

This applies to all actions added or updated — including new ones introduced during fixes or features.
