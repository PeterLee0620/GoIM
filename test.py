import json, random, time
from locust import User, task, events
from websocket import create_connection

class WSUser(User):
    abstract = True          # 告诉 Locust 这不是 HTTP 用户

class ChatUser(WSUser):
    wait_time = lambda _: random.uniform(0.5, 2)

    @task(1)
    def connect_and_chat(self):
        start = time.time()
        try:
            ws = create_connection("ws://localhost:3000/connect", timeout=10)
            hello = ws.recv()
            assert hello == "HELLO"

            uid = "0x" + "".join(random.choices("0123456789abcdef", k=40))
            ws.send(json.dumps({"ID": uid, "Name": "User"}))

            welcome = ws.recv()
            assert welcome.startswith("WELCOME")

            rt = int((time.time() - start) * 1000)
            events.request.fire(
                request_type="WS",
                name="/connect",
                response_time=rt,
                response_length=len(welcome),
                exception=None
            )
            ws.close()

        except Exception as e:
            rt = int((time.time() - start) * 1000)
            events.request.fire(
                request_type="WS",
                name="/connect",
                response_time=rt,
                response_length=0,
                exception=e
            )