import requests
import os
import time
import base64

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

def upload_image():
    filename = input("Enter the image filename to upload: ").strip()
    if not filename:
        print("Error: No filename provided")
        return
    
    if not os.path.exists(filename):
        print(f"Error: File '{filename}' not found")
        return
    
    # Step 1: Upload the image
    print("Uploading image...")
    upload_url = "https://api.canva.com/rest/v1/asset-uploads"
    
    # Get just the filename without path for the asset name
    asset_name = os.path.basename(filename)
    name_base64 = base64.b64encode(asset_name.encode('utf-8')).decode('utf-8')
    
    headers = {
        "Authorization": f"Bearer {ACCESS_TOKEN}",
        "Content-Type": "application/octet-stream",
        "Asset-Upload-Metadata": f'{{"name_base64": "{name_base64}"}}'
    }
    
    with open(filename, 'rb') as f:
        response = requests.post(upload_url, headers=headers, data=f.read())
    
    if response.status_code != 200:
        print(f"Error uploading image: {response.status_code} - {response.text}")
        return
    
    upload_data = response.json()
    job_id = upload_data['job']['id']
    print(f"Image upload job created with ID: {job_id}")
    
    # Poll for upload completion
    print("Waiting for upload to complete...")
    max_attempts = 30
    attempts = 0
    asset_id = None
    
    while attempts < max_attempts:
        job_status_url = f"https://api.canva.com/rest/v1/asset-uploads/{job_id}"
        job_response = requests.get(job_status_url, headers={"Authorization": f"Bearer {ACCESS_TOKEN}"})
        
        if job_response.status_code != 200:
            print(f"Error checking upload status: {job_response.status_code} - {job_response.text}")
            return
        
        job_data = job_response.json()
        status = job_data['job']['status']
        
        if status == "success":
            asset_id = job_data['job']['asset']['id']
            print(f"Upload completed! Asset ID: {asset_id}")
            break
        elif status == "failed":
            print("Upload failed:", job_data['job'].get('error', 'Unknown error'))
            return
        else:
            attempts += 1
            time.sleep(1)
    
    if not asset_id:
        print("Upload timed out")
        return
    
    # Step 2: Create a YouTube thumbnail design
    print("Creating YouTube thumbnail design...")
    design_url = "https://api.canva.com/rest/v1/designs"
    design_data = {
        "design_type": {
            "type": "preset",
            "name": "doc"
        },
        "title": "YouTube Thumbnail",
        "asset_id": asset_id
    }
    
    response = requests.post(design_url, headers={**headers, "Content-Type": "application/json"}, json=design_data)
    
    if response.status_code != 200:
        print(f"Error creating design: {response.status_code} - {response.text}")
        return
    
    design_response = response.json()
    design_id = design_response['design']['id']
    print(f"YouTube thumbnail design created with ID: {design_id}")
    
    # Step 3: Add the uploaded image to the design
    print("Adding image to design...")
    add_element_url = f"https://api.canva.com/rest/v1/designs/{design_id}/elements"
    element_data = {
        "element": {
            "type": "image",
            "asset_id": asset_id
        }
    }
    
    response = requests.post(add_element_url, headers={**headers, "Content-Type": "application/json"}, json=element_data)
    
    if response.status_code != 200:
        print(f"Error adding image to design: {response.status_code} - {response.text}")
        return
    
    print("Image successfully added to YouTube thumbnail design!")
    print(f"Design ID: {design_id}")
    print(f"You can now edit this design in Canva or download it using option 1")

def main_menu():
    if not ACCESS_TOKEN:
        print("Error: CANVA_ACCESS_TOKEN environment variable not set")
        return
    
    while True:
        print("\nCanva Design Tool")
        print("=================")
        print("1. Download design")
        print("2. List designs")
        print("3. Upload image and create YouTube thumbnail")
        print("4. Exit")
        
        choice = input("\nEnter your choice (1-4): ").strip()
        
        if choice == "1":
            download_design()
        elif choice == "2":
            list_designs()
        elif choice == "3":
            upload_image()
        elif choice == "4":
            print("Goodbye!")
            break
        else:
            print("Invalid choice. Please enter 1, 2, 3, or 4.")

if __name__ == "__main__":
    main_menu()

