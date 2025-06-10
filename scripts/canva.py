import requests
import os

ACCESS_TOKEN = os.getenv("CANVA_ACCESS_TOKEN")
DESIGN_ID = os.getenv("CANVA_DESIGN_ID")

# Step 1: Create an export job for PNG
export_url = "https://api.canva.com/rest/v1/exports"
headers = {
    "Authorization": f"Bearer {ACCESS_TOKEN}",
    "Content-Type": "application/json"
}
data = {
    "design_id": DESIGN_ID,
    "format": {
        "type": "png"
    }
}

response = requests.post(export_url, headers=headers, json=data)
if response.status_code != 202:
    print("Error creating export job:", response.text)
    exit()

export_job = response.json()
export_id = export_job["export_id"]

# Step 2: Poll the export job until it's complete
get_job_url = f"https://api.canva.com/rest/v1/exports/{export_id}"
while True:
    job_response = requests.get(get_job_url, headers={"Authorization": f"Bearer {ACCESS_TOKEN}"})
    job = job_response.json()
    status = job["status"]
    if status == "COMPLETED":
        download_url = job["download_url"]
        break
    elif status in ["FAILED", "CANCELLED"]:
        print("Export job failed or was cancelled:", job)
        exit()
    # Wait before polling again
    import time; time.sleep(1)

# Step 3: Download the PNG file
download_response = requests.get(download_url)
with open("design.png", "wb") as f:
    f.write(download_response.content)
print("PNG downloaded as design.png")

