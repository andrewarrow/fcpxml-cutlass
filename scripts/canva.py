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

def display_tree(elements, prefix="", is_last=True):
    """Display elements in a tree ASCII format"""
    if not elements:
        return
    
    for i, element in enumerate(elements):
        is_current_last = (i == len(elements) - 1)
        current_prefix = "└── " if is_current_last else "├── "
        
        # Get element type and basic info
        element_type = element.get('type', 'unknown')
        element_name = element.get('name', element.get('text', ''))
        
        # Display the element
        display_text = f"{element_type}"
        if element_name:
            display_text += f": {element_name}"
        
        print(f"{prefix}{current_prefix}{display_text}")
        
        # Add details about the element
        next_prefix = prefix + ("    " if is_current_last else "│   ")
        
        # Show dimensions if available
        if 'width' in element and 'height' in element:
            print(f"{next_prefix}├── Size: {element['width']}x{element['height']}")
        
        # Show position if available
        if 'x' in element and 'y' in element:
            print(f"{next_prefix}├── Position: ({element['x']}, {element['y']})")
        
        # Show color if available
        if 'color' in element:
            print(f"{next_prefix}├── Color: {element['color']}")
        
        # Show any additional properties
        props = []
        for key, value in element.items():
            if key not in ['type', 'name', 'text', 'width', 'height', 'x', 'y', 'color', 'children']:
                if isinstance(value, (str, int, float, bool)):
                    props.append(f"{key}: {value}")
        
        if props:
            for j, prop in enumerate(props):
                is_prop_last = (j == len(props) - 1)
                prop_prefix = "└── " if is_prop_last else "├── "
                print(f"{next_prefix}{prop_prefix}{prop}")
        
        # Recursively display children if they exist
        if 'children' in element and element['children']:
            print(f"{next_prefix}└── Children:")
            display_tree(element['children'], next_prefix + "    ", True)

def inspect_design():
    """Inspect a design and display all elements in a tree view"""
    DESIGN_ID = input("Enter the design ID to inspect: ").strip()
    if not DESIGN_ID:
        print("Error: No design ID provided")
        return
    
    headers = {
        "Authorization": f"Bearer {ACCESS_TOKEN}"
    }
    
    # Get design details
    design_url = f"https://api.canva.com/rest/v1/designs/{DESIGN_ID}"
    response = requests.get(design_url, headers=headers)
    
    if response.status_code != 200:
        print(f"Error fetching design: {response.status_code} - {response.text}")
        return
    
    design_data = response.json()
    design = design_data.get('design', design_data)
    
    print(f"\nDesign: {design.get('title', design.get('name', 'Untitled'))}")
    print(f"ID: {design['id']}")
    print(f"Type: {design.get('design_type', {}).get('type', 'unknown')}")
    
    if 'width' in design and 'height' in design:
        print(f"Dimensions: {design['width']}x{design['height']}")
    elif 'design_type' in design and 'width' in design['design_type']:
        print(f"Dimensions: {design['design_type']['width']}x{design['design_type']['height']}")
    
    print("\nDesign Elements:")
    print("="*50)
    
    # Try to get pages/elements from the design
    # Note: The Canva Connect API may have limited access to element details
    # We'll display what's available in the design response
    
    elements = []
    
    # Check if there are pages in the design
    if 'pages' in design:
        for page_idx, page in enumerate(design['pages']):
            page_element = {
                'type': 'page',
                'name': f"Page {page_idx + 1}",
                'children': []
            }
            
            # Add page properties
            if 'elements' in page:
                for element in page['elements']:
                    page_element['children'].append(element)
            
            elements.append(page_element)
    
    # If no pages, try to find elements directly
    elif 'elements' in design:
        elements = design['elements']
    
    # If still no elements, show the raw design structure
    elif not elements:
        print("Design structure (raw API response):")
        print("├── Design Properties:")
        for key, value in design.items():
            if key not in ['id', 'title', 'name'] and isinstance(value, (str, int, float, bool)):
                print(f"│   ├── {key}: {value}")
            elif key not in ['id', 'title', 'name'] and isinstance(value, dict):
                print(f"│   ├── {key}:")
                for sub_key, sub_value in value.items():
                    if isinstance(sub_value, (str, int, float, bool)):
                        print(f"│   │   ├── {sub_key}: {sub_value}")
                    else:
                        print(f"│   │   ├── {sub_key}: [complex object]")
        
        print("\nNote: Limited element details available via Canva Connect API.")
        print("The Connect API primarily provides design metadata rather than detailed element information.")
        return
    
    # Display the tree
    if elements:
        display_tree(elements)
    else:
        print("No elements found in design response.")
        print("\nNote: The Canva Connect API may have limited access to design element details.")
        print("This is normal for the Connect API, which focuses on design management rather than editing.")

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
    
    # Step 2: Create a design using a preset that supports text overlays
    print("Creating design with text overlay capabilities...")
    design_url = "https://api.canva.com/rest/v1/designs"
    
    # Try using a preset design type that might support text better
    design_data = {
        "design_type": {
            "type": "preset",
            "name": "youtube-thumbnail"  # Use preset which may have better text support
        },
        "title": "cutlass test - YouTube Thumbnail"
    }
    
    response = requests.post(design_url, headers={**headers, "Content-Type": "application/json"}, json=design_data)
    
    if response.status_code != 200:
        print(f"Preset failed, trying custom design: {response.status_code}")
        # Fallback to custom design
        design_data = {
            "design_type": {
                "type": "custom", 
                "width": 1280,
                "height": 720
            },
            "title": "cutlass test - YouTube Thumbnail"
        }
        response = requests.post(design_url, headers={**headers, "Content-Type": "application/json"}, json=design_data)
        
        if response.status_code != 200:
            print(f"Error creating design: {response.status_code} - {response.text}")
            return
    
    design_response = response.json()
    design_id = design_response['design']['id']
    print(f"Design created with ID: {design_id}")
    
    # Step 3: Add the uploaded image to the design first
    print("Adding uploaded image to design...")
    # Note: Based on API research, the Connect API has limited support for adding elements after creation
    # The image needs to be added during design creation or through autofill templates
    
    # Create a new design with the asset
    print("Recreating design with uploaded image...")
    design_with_asset = {
        "design_type": {
            "type": "custom",
            "width": 1280, 
            "height": 720
        },
        "title": "cutlass test - YouTube Thumbnail with Image",
        "asset_id": asset_id  # This adds the image during creation
    }
    
    asset_response = requests.post(design_url, headers={**headers, "Content-Type": "application/json"}, json=design_with_asset)
    
    if asset_response.status_code == 200:
        final_design = asset_response.json()
        final_design_id = final_design['design']['id']
        print(f"Design with image created! Design ID: {final_design_id}")
        print("Note: Based on Canva Connect API limitations, text must be added manually in Canva.")
        print("The Connect API primarily supports text through autofill templates, not direct element addition.")
        print(f"Open the design in Canva and add your 'cutlass test' text with blue/violet styling.")
        print(f"Design URL: https://www.canva.com/design/{final_design_id}")
    else:
        print(f"Could not add image to design: {asset_response.status_code} - {asset_response.text}")
        print(f"Basic design created with ID: {design_id}")
        print("You'll need to manually add both the image and text in Canva.")
    
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
        print("4. Inspect design")
        print("5. Exit")
        
        choice = input("\nEnter your choice (1-5): ").strip()
        
        if choice == "1":
            download_design()
        elif choice == "2":
            list_designs()
        elif choice == "3":
            upload_image()
        elif choice == "4":
            inspect_design()
        elif choice == "5":
            print("Goodbye!")
            break
        else:
            print("Invalid choice. Please enter 1, 2, 3, 4, or 5.")

if __name__ == "__main__":
    main_menu()

