import requests
import os
import time

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
if response.status_code not in [200, 202]:
    print("Error creating export job:", response.text)
    exit()

export_job = response.json()
job = export_job["job"]
export_id = job["id"]

print(f"Export job created with ID: {export_id}, status: {job['status']}")

# Step 2: Poll the export job until it's complete
get_job_url = f"https://api.canva.com/rest/v1/exports/{export_id}"
max_attempts = 60  # Maximum 60 attempts (1 minute with 1-second intervals)
attempts = 0

while attempts < max_attempts:
    job_response = requests.get(get_job_url, headers={"Authorization": f"Bearer {ACCESS_TOKEN}"})
    if job_response.status_code != 200:
        print(f"Error polling job status: {job_response.status_code} - {job_response.text}")
        exit()
    
    job_data = job_response.json()
    job = job_data.get("job", job_data)  # Handle both wrapped and unwrapped responses
    status = job["status"]
    
    print(f"Job status: {status}")
    
    if status == "completed":
        download_url = job["download_url"]
        break
    elif status in ["failed", "cancelled"]:
        print("Export job failed or was cancelled:", job)
        exit()
    elif status == "in_progress":
        attempts += 1
        time.sleep(1)
    else:
        print(f"Unknown status: {status}")
        attempts += 1
        time.sleep(1)

if attempts >= max_attempts:
    print("Timeout: Export job took too long to complete")
    exit()

# Step 3: Download the PNG file
print(f"Downloading from: {download_url}")
download_response = requests.get(download_url)
if download_response.status_code != 200:
    print(f"Error downloading file: {download_response.status_code} - {download_response.text}")
    exit()

with open("design.png", "wb") as f:
    f.write(download_response.content)
print("PNG downloaded as design.png")

