---
title: "Quick start"
description: "Run your first sinafinance command."
weight: 30
---

Once `sinafinance` is on your `PATH`:

```bash
sinafinance --help       # see the command tree
sinafinance version      # build info
```

This is a fresh scaffold, so the command tree is just `version` for now. Add
your first real command in `cli/`, build on the `sinafinance-cli` library package,
and document it here.

A good first command usually fetches one thing and prints it as JSON, so the
output pipes straight into `jq` and the rest of your tools.
