# go-ssh-example

My first attempt to use SSH in Go (yes, it works)

Run as `go build && ./go-ssh-example <yaml-files...>`.

For such a "script"

```
# sample config
command: cd /tmp && pwd; tty
tty: false
domain: example.com
hosts:
        - test-web
        - test-vip
        - test-whois
# EOF #
```

Sample output one may expect looks like this one:

```
2020/07/23 16:14:13 INFO: [0] @"test-web.example.com": "cd /tmp && pwd; tty"
2020/07/23 16:14:13 INFO: [1] @"test-vip.example.com": "cd /tmp && pwd; tty"
2020/07/23 16:14:13 INFO: [2] @"test-whois.example.com": "cd /tmp && pwd; tty"
2020/07/23 16:14:14 WARNING: [2] @"test-whois.example.com": "/tmp\nnot a tty\n" (Process exited with status 1) 492.249924ms
2020/07/23 16:14:14 WARNING: [1] @"test-vip.example.com": "/tmp\nnot a tty\n" (Process exited with status 1) 504.932858ms
2020/07/23 16:14:14 WARNING: [0] @"test-web.example.com": "/tmp\nnot a tty\n" (Process exited with status 1) 507.690531ms
2020/07/23 16:14:14 INFO: Total run time 1.504873313s for 3 tasks in 508.72044ms (3.0× speedup)
2020/07/23 16:14:14 WARNING: There were 3 failed tasks out of 3, 100%
```

One may change the `tty` value to `true` to get rid of those errors.

# EOF #
