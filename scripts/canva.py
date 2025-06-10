import requests
import os
import time

ACCESS_TOKEN = os.getenv("CANVA_ACCESS_TOKEN")

def download_design():
    DESIGN_ID = input("Enter the design ID to download: ").strip()
    if not DESIGN_ID:
        print("Error: No design ID provided")
        return
    
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
        return

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
            return
        
        job_data = job_response.json()
        job = job_data.get("job", job_data)  # Handle both wrapped and unwrapped responses
        status = job["status"]
        
        print(f"Job status: {status}")
        
        if status == "success":
            print(job)
            download_url = job["urls"][0]
            break
        elif status in ["failed", "cancelled"]:
            print("Export job failed or was cancelled:", job)
            return
        elif status == "in_progress":
            attempts += 1
            time.sleep(1)
        else:
            print(f"Unknown status: {status}")
            attempts += 1
            time.sleep(1)

    if attempts >= max_attempts:
        print("Timeout: Export job took too long to complete")
        return

    # Step 3: Download the PNG file
    print(f"Downloading from: {download_url}")
    download_response = requests.get(download_url)
    if download_response.status_code != 200:
        print(f"Error downloading file: {download_response.status_code} - {download_response.text}")
        return

    filename = f"{DESIGN_ID}.png"
    with open(filename, "wb") as f:
        f.write(download_response.content)
    print(f"PNG downloaded as {filename}")

def list_designs():
    headers = {
        "Authorization": f"Bearer {ACCESS_TOKEN}"
    }
    
    # Get user's designs
    designs_url = "https://api.canva.com/rest/v1/designs"
    response = requests.get(designs_url, headers=headers)
    
    if response.status_code != 200:
        print(f"Error fetching designs: {response.status_code} - {response.text}")
        return
    
    designs_data = response.json()
    designs = designs_data.get("items", [])
    
    if not designs:
        print("No designs found")
        return
    
    print(f"Found {len(designs)} designs:")
    print("-" * 50)
    
    for i, design in enumerate(designs, 1):
        title = design.get('title', design.get('name', 'Untitled'))
        print(f"{i}. {title}")
        print(f"   ID: {design['id']}")
        if 'created_at' in design:
            print(f"   Created: {design['created_at']}")
        elif 'created' in design:
            print(f"   Created: {design['created']}")
        print()

def main_menu():
    if not ACCESS_TOKEN:
        print("Error: CANVA_ACCESS_TOKEN environment variable not set")
        return
    
    while True:
        print("\nCanva Design Tool")
        print("=================")
        print("1. Download design")
        print("2. List designs")
        print("3. Exit")
        
        choice = input("\nEnter your choice (1-3): ").strip()
        
        if choice == "1":
            download_design()
        elif choice == "2":
            list_designs()
        elif choice == "3":
            print("Goodbye!")
            break
        else:
            print("Invalid choice. Please enter 1, 2, or 3.")

if __name__ == "__main__":
    main_menu()

