#!/usr/bin/env python3

import os
import subprocess
import asyncio
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

async def login_to_canva():
    """Login to Canva using email, password, and 2FA"""
    canva_email = os.getenv('CANVA_EMAIL')
    canva_pass = os.getenv('CANVA_PASS')
    
    if not canva_email or not canva_pass:
        raise ValueError("CANVA_EMAIL and CANVA_PASS environment variables are required")
    
    async with async_playwright() as p:
        # Launch headless browser
        browser = await p.chromium.launch(headless=True)
        context = await browser.new_context()
        page = await context.new_page()
        
        try:
            # Navigate to the Canva design URL
            print("Navigating to Canva design...")
            await page.goto("https://www.canva.com/design/DAGp4ARj9e0")
            
            # Wait for login button and click it
            print("Looking for login button...")
            await page.wait_for_selector('[data-testid="login-button"]', timeout=10000)
            await page.click('[data-testid="login-button"]')
            
            # Fill in email
            print("Entering email...")
            await page.wait_for_selector('input[type="email"]', timeout=10000)
            await page.fill('input[type="email"]', canva_email)
            
            # Click continue/next button
            await page.click('button[type="submit"]')
            
            # Fill in password
            print("Entering password...")
            await page.wait_for_selector('input[type="password"]', timeout=10000)
            await page.fill('input[type="password"]', canva_pass)
            
            # Click login button
            await page.click('button[type="submit"]')
            
            # Wait for 2FA prompt
            print("Waiting for 2FA prompt...")
            await page.wait_for_selector('input[type="text"][placeholder*="code"]', timeout=10000)
            
            # Generate and enter TOTP code
            print("Generating 2FA code...")
            totp_code = await get_totp_code()
            print(f"Entering 2FA code: {totp_code}")
            await page.fill('input[type="text"][placeholder*="code"]', totp_code)
            
            # Submit 2FA
            await page.click('button[type="submit"]')
            
            # Wait for successful login (check for design interface)
            print("Waiting for login to complete...")
            await page.wait_for_selector('[data-testid="design-canvas"]', timeout=15000)
            
            print("Successfully logged into Canva!")
            
            # Keep the page open for a moment to ensure login is complete
            await page.wait_for_timeout(2000)
            
        except Exception as e:
            print(f"Error during login process: {e}")
            raise
        finally:
            await browser.close()

async def main():
    """Main function to run the Canva login automation"""
    try:
        await login_to_canva()
    except Exception as e:
        print(f"Login failed: {e}")
        return 1
    return 0

if __name__ == "__main__":
    asyncio.run(main())