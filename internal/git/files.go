package git

const (
	HashRegexp = "[0-9a-f]{40}"
	PathRoot   = ".git"
	PathIndex  = "index"
	PathHead   = "HEAD"
	PrefixRef  = "ref: "
)

func getPathsRef() []string {
	return []string{
		"FETCH_HEAD",
		"logs/stash",
		"logs/HEAD",
		"logs/refs/heads/master",
		"logs/refs/heads/main",
		"logs/refs/heads/origin",
		"logs/refs/remotes/origin/HEAD",
		"logs/refs/remotes/origin/master",
		"logs/refs/remotes/origin/main",
		"packed-refs",
		"refs/heads/master",
		"refs/heads/main",
		"refs/heads/origin",
		"refs/remotes/origin/master",
		"refs/remotes/origin/main",
	}
}

func getPathsCommon() []string {
	return []string{
		"ORIG_HEAD",
		"application",
		"description",
		"COMMIT_EDITMSG",
		"config",
		"info/exclude",
		"refs/remotes/origin/HEAD",

		"hooks/applypatch-msg",
		"hooks/applypatch-msg.sample",
		"hooks/commit-msg",
		"hooks/commit-msg.sample",
		"hooks/fsmonitor-watchman",
		"hooks/fsmonitor-watchman.sample",
		"hooks/post-commit",
		"hooks/post-commit.sample",
		"hooks/post-receive",
		"hooks/post-receive.sample",
		"hooks/post-update",
		"hooks/post-update.sample",
		"hooks/pre-applypatch",
		"hooks/pre-applypatch.sample",
		"hooks/pre-commit",
		"hooks/pre-commit.sample",
		"hooks/pre-merge-commit",
		"hooks/pre-merge-commit.sample",
		"hooks/pre-push",
		"hooks/pre-push.sample",
		"hooks/pre-rebase",
		"hooks/pre-rebase.sample",
		"hooks/pre-receive",
		"hooks/pre-receive.sample",
		"hooks/prepare-commit-msg",
		"hooks/prepare-commit-msg.sample",
		"hooks/push-to-checkout",
		"hooks/push-to-checkout.sample",
		"hooks/sendemail-validate",
		"hooks/sendemail-validate.sample",
		"hooks/update",
		"hooks/update.sample",
	}
}
