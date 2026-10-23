import json
import os
import random
from datetime import datetime, timezone
from locust import HttpUser, task, between

CARNET = "201901657"
CFG_PATH = os.path.join(os.path.dirname(__file__), "config.json")


try:
    with open(CFG_PATH) as f:
        CFG = json.load(f)
except FileNotFoundError:
    CFG = {"countries": ["USA", "RUS", "CHN", "ESP", "GTM"], "planes_max": 50, "ships_max": 30}


class ReportUser(HttpUser):
    wait_time = between(0.01, 0.1)

    @task
    def send_report(self):
        payload = {
            "country": random.choice(CFG["countries"]),
            "warplanes_in_air": random.randint(0, CFG["planes_max"]),
            "warships_in_water": random.randint(0, CFG["ships_max"]),
            "timestamp": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
        }
        self.client.post(f"/grpc-{CARNET}", json=payload)
