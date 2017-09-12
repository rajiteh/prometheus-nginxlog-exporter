listen {
  port = 4040
}

application "webapp" {
  log_files = [
    "test.log",
    "foo.log"
  ]
  format = "$remote_addr - $remote_user [$time_local] \"$request\" $status $body_bytes_sent \"$http_referer\" \"$http_user_agent\""
  labels {
    type = "magicapp"
    foo = "bar"
  }
}
