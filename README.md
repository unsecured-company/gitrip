# GitRip
Download exposed `.git` repositories from web servers.  
Testing and feedback are welcome, feel free to
[open an issue](https://github.com/unsecured-company/gitrip/issues).

## Installation
Repository contains Makefile which build for all main platforms and architectures.

## Usage
```
# Three main commands
gitrip fetch   #Fetch URL or batch file of URLs
gitrip check   #Check URL or batch file of URLs
gitrip index   #List files from .git/index

# Examples
gitrip check --file domains.txt >valid.txt 2>errors.log
gitrip fetch unsecured.company
gitrip fetch https://unsecured.company/admin/
gitrip index dumps/unsecured.company/.git/index

# Add completion in Bash
gitrip completion bash | sudo tee /etc/bash_completion.d/gitrip > /dev/null
```


## Notes
GitRip does not run `git checkout` automatically after downloading.  
This is intentional for easier cross-platform compatibility and reduce storage usage.  

## TODO
- Add support for a simple `wget`-style download 🙂
- Proxy support
- Detect and handle stale/slow downloads
- Follow redirects and display the final URL
- Identify if HTTP and HTTPS point to the same resource and use only one
- Context, signal catching
- Tag parsing

## Acknowledgments
- [Maxime Arthaud – git-dumper](https://github.com/arthaud/git-dumper)
- [maia arson crimew – Goop](https://github.com/nyancrimew/goop)

## Disclaimer
This tool is for educational and legal use only.  
Do not use it without proper authorization from the data owner.

## Support
If you find GitRip useful, feel free to support my work.

BTC&nbsp;`bc1qv79sm8zp70jsqa4dpweqeg9g2lpyplfszhqzyl`

ETH&nbsp;`0x7A0ac7852258578cc57635206959C848A53413a4`

SOL&nbsp;`C7YKx3AUaqFGA5QafhTy7vQZVtUqiJAUP9N9nzkV2oA9`

XMR&nbsp;`85aHby9N8zRKJFvkR1sEqoAhsq3hm3XpKGNDwEozGhLkN7sfKKMLkx1KdgtxHxmJR44gHmV6MrYZPbgPLQQso4hCKMRVRmE`
