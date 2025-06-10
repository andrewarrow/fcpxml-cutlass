#!/usr/bin/env python3

import os
import subprocess
import asyncio
import json
from playwright.async_api import async_playwright

async def get_totp_code():
    """Generate TOTP code using oathtool"""
    canva_two = os.getenv('CANVA_TWO')
    if not canva_two:
        raise ValueError("CANVA_TWO environment variable not found")
    
    try:
        result = subprocess.run(
            ['oathtool', '--totp', '-b', canva_two],
            capture_output=True,
            text=True,
            check=True
        )
        return result.stdout.strip()
    except subprocess.CalledProcessError as e:
        raise RuntimeError(f"Failed to generate TOTP code: {e}")

async def extract_design_elements():
    """Login to Canva and extract design elements from API responses"""
    canva_email = os.getenv('CANVA_EMAIL')
    canva_pass = os.getenv('CANVA_PASS')
    
    if not canva_email or not canva_pass:
        raise ValueError("CANVA_EMAIL and CANVA_PASS environment variables are required")
    
    design_data = []
    api_responses = []
    
    async with async_playwright() as p:
        # Launch headless browser
        browser = await p.chromium.launch(headless=True)
        context = await browser.new_context()
        page = await context.new_page()
        
        # Intercept network requests to capture API responses
        async def handle_response(response):
            url = response.url
            # Look for API endpoints that might contain design data
            if any(keyword in url.lower() for keyword in ['api', 'graphql', 'design', 'element', 'document']):
                try:
                    if response.status == 200 and 'json' in response.headers.get('content-type', ''):
                        data = await response.json()
                        api_responses.append({
                            'url': url,
                            'data': data
                        })
                        print(f"Captured API response from: {url}")
                except Exception as e:
                    print(f"Failed to parse response from {url}: {e}")
        
        page.on('response', handle_response)
        
        try:
            # Navigate to the Canva design URL
            print("Navigating to Canva design...")
            await page.goto("https://www.canva.com/design/DAGp4ARj9e0")
            
            # Wait for page to load and check if already logged in
            await page.wait_for_timeout(3000)
            
            # Check if we're already at the design (logged in)
            try:
                await page.wait_for_selector('[data-testid="design-canvas"]', timeout=3000)
                print("Already logged in!")
            except:
                print("Not logged in, looking for login options...")
                
                # Look for "Continue with email" button specifically
                email_button_selectors = [
                    'button:has-text("Continue with email")',
                    'button .khPe7Q:text("Continue with email")',
                    'button:text("Continue with email")',
                    'button[type="button"]:has-text("Continue with email")'
                ]
                
                login_clicked = False
                for selector in email_button_selectors:
                    try:
                        print(f"Looking for 'Continue with email' button with selector: {selector}")
                        await page.wait_for_selector(selector, timeout=3000)
                        await page.click(selector)
                        print(f"Clicked 'Continue with email' button")
                        login_clicked = True
                        break
                    except:
                        continue
                
                if not login_clicked:
                    # Try other login options
                    other_login_selectors = [
                        'button:has-text("Log in")',
                        'a:has-text("Log in")',
                        'button:has-text("Sign in")',
                        'a:has-text("Sign in")',
                    ]
                    
                    for selector in other_login_selectors:
                        try:
                            print(f"Trying alternative login selector: {selector}")
                            await page.wait_for_selector(selector, timeout=2000)
                            await page.click(selector)
                            print(f"Clicked login with selector: {selector}")
                            login_clicked = True
                            break
                        except:
                            continue
                
                if not login_clicked:
                    # If no login button found, maybe we need to go to login page directly
                    print("No login button found, navigating to login page...")
                    await page.goto("https://www.canva.com/login")
                    await page.wait_for_timeout(2000)
            
                # Continue with login if not already logged in
                if not await page.locator('[data-testid="design-canvas"]').count():
                    # Fill in email - look for the specific form structure
                    print("Entering email...")
                    email_selectors = [
                        'input[type="text"][inputmode="email"]',
                        'input[name="email"]',
                        'input[autocomplete="email"]',
                        'input.bCVoGQ',
                        'input[type="email"]',
                        '#email'
                    ]
                    email_filled = False
                    for selector in email_selectors:
                        try:
                            await page.wait_for_selector(selector, timeout=5000)
                            await page.fill(selector, canva_email)
                            print(f"Filled email with selector: {selector}")
                            email_filled = True
                            break
                        except:
                            continue
                    
                    if not email_filled:
                        print("Could not find email input field")
                        return []
                    
                    # Click continue/next button - look for the specific button structure
                    submit_selectors = [
                        'button[type="submit"]:has-text("Continue")',
                        'button.a2l15A:has-text("Continue")',
                        'button[type="submit"]',
                        'button:has-text("Continue")',
                        'button:has-text("Next")'
                    ]
                    submit_clicked = False
                    for selector in submit_selectors:
                        try:
                            await page.click(selector)
                            print(f"Clicked continue with selector: {selector}")
                            submit_clicked = True
                            break
                        except:
                            continue
                    
                    if not submit_clicked:
                        print("Could not find continue button")
                        return []
                    
                    # Fill in password - look for the specific password form structure
                    print("Entering password...")
                    password_selectors = [
                        'input[type="password"][autocomplete="current-password"]',
                        'input[type="password"][placeholder="Enter password"]',
                        'input[name="password"]',
                        'input.bCVoGQ[type="password"]',
                        'input[type="password"]',
                        '#password'
                    ]
                    password_filled = False
                    for selector in password_selectors:
                        try:
                            await page.wait_for_selector(selector, timeout=5000)
                            await page.fill(selector, canva_pass)
                            print(f"Filled password with selector: {selector}")
                            password_filled = True
                            break
                        except:
                            continue
                    
                    if not password_filled:
                        print("Could not find password input field")
                        return []
                    
                    # Click login button - look for the specific login button structure
                    login_submit_selectors = [
                        'button[type="submit"]:has-text("Log in")',
                        'button._KubKw:has-text("Log in")',
                        'button[type="submit"]',
                        'button:has-text("Log in")',
                        'button:has-text("Sign in")'
                    ]
                    login_submit_clicked = False
                    for selector in login_submit_selectors:
                        try:
                            await page.click(selector)
                            print(f"Clicked login submit with selector: {selector}")
                            login_submit_clicked = True
                            break
                        except:
                            continue
                    
                    if not login_submit_clicked:
                        print("Could not find login submit button")
                        return []
                    
                    # Wait for 2FA prompt - look for the specific authenticator form structure
                    print("Waiting for 2FA prompt...")
                    totp_selectors = [
                        'input[type="text"][inputmode="numeric"][placeholder="Enter code"]',
                        'input[type="text"][maxlength="6"][placeholder="Enter code"]',
                        'input.ztpQdA[type="text"]',
                        'input[type="text"][pattern="\\d*"]',
                        'input[placeholder="Enter code"]',
                        'input[type="text"][placeholder*="code"]',
                        'input[placeholder*="verification"]',
                        'input[name*="code"]',
                        'input[name*="otp"]'
                    ]
                    totp_input = None
                    for selector in totp_selectors:
                        try:
                            await page.wait_for_selector(selector, timeout=5000)
                            totp_input = selector
                            print(f"Found 2FA input with selector: {selector}")
                            break
                        except:
                            continue
                    
                    if totp_input:
                        # Generate and enter TOTP code
                        print("Generating 2FA code...")
                        totp_code = await get_totp_code()
                        print(f"Entering 2FA code: {totp_code}")
                        await page.fill(totp_input, totp_code)
                        
                        # Submit 2FA - look for the specific Continue button structure
                        totp_submit_selectors = [
                            'button[type="submit"]:has-text("Continue")',
                            'button.ak_kbw:has-text("Continue")',
                            'button[type="submit"]',
                            'button:has-text("Continue")',
                            'button:has-text("Verify")'
                        ]
                        totp_submit_clicked = False
                        for selector in totp_submit_selectors:
                            try:
                                await page.click(selector)
                                print(f"Clicked 2FA submit with selector: {selector}")
                                totp_submit_clicked = True
                                break
                            except:
                                continue
                        
                        if not totp_submit_clicked:
                            print("Could not find 2FA submit button")
                            return []
                    else:
                        print("Could not find 2FA input field")
                        return []
                    
                    # Wait for successful login (check for design interface)
                    print("Waiting for login to complete...")
                    await page.wait_for_selector('[data-testid="design-canvas"]', timeout=15000)
                    
                    print("Successfully logged into Canva!")
                else:
                    print("Already at design interface!")
            
            # Wait longer for design data to load
            print("Waiting for design data to load...")
            await page.wait_for_timeout(5000)
            
            # Try to trigger additional API calls by interacting with the design
            try:
                # Look for pages/slides in the design
                pages_selector = '[data-testid="page-thumbnail"], [data-testid="slide-thumbnail"]'
                await page.wait_for_selector(pages_selector, timeout=5000)
                print("Found design pages/slides")
            except:
                print("No pages/slides found or timeout")
            
            # Wait a bit more for any additional API calls
            await page.wait_for_timeout(3000)
            
            print(f"\nCaptured {len(api_responses)} API responses")
            
            # Analyze the captured API responses for design elements
            for i, response in enumerate(api_responses):
                print(f"\n--- API Response {i+1} ---")
                print(f"URL: {response['url']}")
                
                # Look for design-related data
                data = response['data']
                if isinstance(data, dict):
                    # Look for common design data patterns
                    if 'design' in data:
                        print("Found design data:")
                        print(json.dumps(data['design'], indent=2)[:1000] + "..." if len(json.dumps(data['design'])) > 1000 else json.dumps(data['design'], indent=2))
                    
                    if 'elements' in data:
                        print("Found elements data:")
                        print(json.dumps(data['elements'], indent=2)[:1000] + "..." if len(json.dumps(data['elements'])) > 1000 else json.dumps(data['elements'], indent=2))
                    
                    if 'pages' in data:
                        print("Found pages data:")
                        print(json.dumps(data['pages'], indent=2)[:1000] + "..." if len(json.dumps(data['pages'])) > 1000 else json.dumps(data['pages'], indent=2))
                    
                    # Look for any data containing our design ID
                    if 'DAGp4ARj9e0' in json.dumps(data):
                        print("Found data with design ID DAGp4ARj9e0:")
                        print(json.dumps(data, indent=2)[:2000] + "..." if len(json.dumps(data)) > 2000 else json.dumps(data, indent=2))
            
        except Exception as e:
            print(f"Error during process: {e}")
            raise
        finally:
            await browser.close()
    
    return api_responses

async def main():
    """Main function to run the Canva design extraction"""
    try:
        api_responses = await extract_design_elements()
        print(f"\nExtraction complete. Captured {len(api_responses)} API responses with design data.")
    except Exception as e:
        print(f"Extraction failed: {e}")
        return 1
    return 0

if __name__ == "__main__":
    asyncio.run(main())