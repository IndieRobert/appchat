Appchat

# Installation

Tested under windows 10 + recent Chrome.

## Start backend server

> cd appchat/sv

> go run sv.go

Server is starting up.

## Start webserver

> cd appchat

> python -m SimpleHTTPServer 8000

## Chat in your browser

go to http://localhost:8000/

Enter your username and click on "chat now!"

Your should see the chatroom now

Type your messages and click send

Open as many tab as you want to test chatting with multiple clients

## Bots

'bots.go' allow the creation of hundreds of clients in order to stress test the system

> cd appchat/sv

> go run bots.go

Look at the ping and at your CPU load now, to see the impact of number of clients

# Acknowledgement of & proposed solutions for any deficiencies.

  * use https and wws
  * use tokens (uuid) to uniquely identify user and put them in a database
  * put sv.go and bots.go into src folder with a proper package
  * Using bots (see bots.go) and the ping command, 350 clients sending one message per second make my laptop CPU run at 100% and the ping grows dramatically. The quick guess is that the message broadcasting goes mad.

  If the message broadcasting is slow, one solution would be to batch messages together according to a time window. For example, instead of broadcasting 1 message at a time, the server would wait say 200ms, take all the messages that need to be dispatched, zip them into a blob, and broadcast that to the clients.

  UPDATE: I have implemented the batching mechanism. With a time window of 250ms, 350+ bots take now 25% CPU. It takes 900+ bots to have CPU at 100% and for some reasons the ping is still under 1 second - ish. 

  The user is not much impacted by the maximum of 250ms latency added by the batching time window but still when there is no traffic, it should be instant. So an improvement could be that the server would size the time window according to its CPU load / latency.

  I'm not satisfied actually with the results because 1) I was expecting a greater number of bots for 100% CPU, 2) the ping is oscillating because 3) server is sleeping most of the time.

  * [BUG] If you block the execution of the server (under windows its possible by blocking the console), then things goes a bit mad.
  * [BUG] when forcefully quitting bots.go program, only a few clients are disconnected from the server point of view.
  * [BUG] in some case, when a browser page disconnect for example, the bots seems to not be able to 'claimUserName' anymore
  * Implement codec instead of using "variant" + BFS (Big Fucking Switch) for client/server communication.
  * Fix the "websocket connection get closed when flow quit handler" problem.
  * This is my first program in go, there must be some big mistakes go-wise.
  * Server shutdown is missing
  * We want to be able to turn on/off logging; right now, some are commented out for performances reasons.
  * bots.go is super quick&dirty with dedupe code and everything...
  * batching mechanism 250ms window is actually too big, user feel the latency.
  * Find a better way to manage js UI, using ReactJS?