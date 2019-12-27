# go-ssh-example

My first attempt to use SSH in Go (yes, it works)

Run as `go build && ./go-ssh-example <host-where-you-have-access-to> [<command-to-run-there>]`.

Default command is `pwd` (safe enough).

Sample output one may expect looks like this one:

```
2019/12/27 12:12:44 INFO: started
2019/12/27 12:12:44 NOTE: Log level INFO -> DEBUG
2019/12/27 12:12:44 INFO: Running as "jno" (jno)
2019/12/27 12:12:44 INFO: Loaded "/home/jno/.ssh/config"
2019/12/27 12:12:44 INFO: SSH config file "/home/jno/.ssh/config" has 46 host entries
2019/12/27 12:12:44 DEBUG: "/home/jno/.ssh/config"["host.example.com"]["Port"] ("22") = "22"
2019/12/27 12:12:44 DEBUG: "/home/jno/.ssh/config"["host.example.com"]["User"] ("jno") = "root"
2019/12/27 12:12:44 DEBUG: host keys in "/home/jno/.ssh/known_hosts"
2019/12/27 12:12:44 DEBUG: Found key for host "host.example.com"
2019/12/27 12:12:44 INFO: Running "pwd"
2019/12/27 12:12:44 INFO: Got "/root\n"
2019/12/27 12:12:44 INFO: stopped
```

One may add the `-term` flag and run, say, `tty` command to see the difference.

# EOF #
