import random
from datetime import datetime, timezone
from locust import HttpUser, task, between

COUNTRIES = ["USA", "RUS", "CHN", "ESP", "GTM"]
CARNET = "201901657"


class ReportUser(HttpUser):
    wait_time = between(0.01, 0.1)

    @task
    def send_report(self):
        payload = {
            "country": random.choice(COUNTRIES),
            "warplanes_in_air": random.randint(0, 50),
            "warships_in_water": random.randint(0, 30),
            "timestamp": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
        }
        self.client.post(f"/grpc-{CARNET}", json=payload)
