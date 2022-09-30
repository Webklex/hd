# HD - Host header ding

Check a single or many targets how they behave if an altered host header is supplied.


## Installation
```bash
go install -v github.com/webklex/hd@main
```

## Usage
```bash
Usage of hd:
  --output    string    File to store all outputs
  --target    string    Targets to scan
  --target-file string  File containing a list of targets
  --user-agent  string  Set a custom user agent (default "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/105.0.0.0 Safari/537.36")
  --host-name string    Fake hostname used to verify host header injection (default "somethingbadthatdoesntexist-hopefully.com")
  --scheme    string    Default url scheme (default "https")
  --score     float     Percentage of response lines that have to be identical (default 90)
  --threads   int       Number of threads (default 10)
  --delay     duration  Delay between requests (default 0s)
  --timeout   duration  Request timeout (default 10s)
  --redirects Follow all redirects
  --no-color  Disable color output
  --version   Show version and exit
```

```bash
./hd --target rediit.com,google.com,twitter.com,doesntexist.com --host-name evil.com
```

```text
[success] https://reddit.com [200] [7201] [200] [5050] [22.17] [redirect]
[info] https://reddit.com [200] [7201] [200] [5050] [22.17] [none]
[success] https://twitter.com [200] [6023] [200] [6023] [100.00] [injection]
[failed] https://google.com [301] [220] [404] [1561] [14.29] []
[error] host unreachable https://doesntexist.com
```

The default output is grepable. If you don't like color, you can disable it with `--no-color`.
`[status] target [request status code] [request response size] [modified request status code] [modified request response size] [response difference score] [finding]`

**Finding:**
- `injection` -- the faked host header seems to be accepted
- `recirect` -- the faked host header redirected to the fake host
- `none` -- the second response deviates too much from the defined score
- ` ` -- it`s not clear what's exactly happening or what it could mean

**Status:**
- `success` -- looks good :)
- `info` -- probably nothing, but who knows
- `failed` -- Initial request isn't returning status `200 OK`
- `error` -- something bad has happened

The supplied targets don't have to be an url. A plain domain name works as well. 
The default schema `https` will get applied to any target without one. You can change this by providing the following 
argument: `--scheme http`


## Build
```bash
go build -a -ldflags "-w -s -X main.buildVersion=custom" -o hd
```


## Security
If you discover any security related issues, please email github@webklex.com instead of using the issue tracker.


## Credits
- [Webklex][link-author]
- [All Contributors][link-contributors]


## License
The MIT License (MIT). Please see [License File](LICENSE.md) for more information.


[link-author]: https://github.com/webklex
[link-contributors]: https://github.com/webklex/hd/graphs/contributors