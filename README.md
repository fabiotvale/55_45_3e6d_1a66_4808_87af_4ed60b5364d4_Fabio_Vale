To view the available command options and default values:

> go run main.go -h

Example:

> go run main.go -url=https://postman-echo.com/post -key=123 -rqs=50 -duration=60 -verbose

Tips:

> It's not recommended to set verbose=true in case you are using a high RQS value (RQS > 50 or so)
> Also do not set verbose=true in case you define a long duration for the program execution

Considerations:

- I've added some extra execution flags:

1. duration: it comes in handy if you want to define a time-boxed loop of X RQS
2. verbose: for debugging purposes you may see the results of each and all requests

- I've chosen golang because it makes it extremely simple to handle concurrency (which is the main requirement in this project) when compared to other languages

- I'm using a channel to receive the results of each request in a synchronous way, and once it's finished I can compile them into a single digest report

- What I would add to it in the future:

1. a new flag to preset the kind of performance test you want to do (smoke, load, stress) instead of only allowing RPS as a param
2. in the digest report I would group the results based on http status code and error messages (and the amount of times each of them has happened); this way we would have a clear map of the kind of error that happen the most
3. have a richer report including: average response time, request duration split into its different steps (sending, receiving, handshaking, waiting, etc), data received/sent, among other relevant information to perform a full health check of an endpoint
