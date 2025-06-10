#!/usr/bin/env python3
import asyncio
import re
from playwright.async_api import async_playwright

async def scrape_youtube_trending():
    async with async_playwright() as p:
        browser = await p.chromium.launch(headless=True)
        page = await browser.new_page()
        
        print("DEBUG: Navigating to YouTube trending page...")
        await page.goto('https://www.youtube.com/feed/trending')
        await page.wait_for_load_state('networkidle')
        print("DEBUG: Page loaded successfully")
        
        video_elements = await page.query_selector_all('a[href*="/watch?v="]')
        print(f"DEBUG: Found {len(video_elements)} video link elements")
        
        videos = []
        processed_count = 0
        for i, element in enumerate(video_elements):
            href = await element.get_attribute('href')
            
            if href:
                video_id_match = re.search(r'v=([a-zA-Z0-9_-]+)', href)
                if video_id_match:
                    video_id = video_id_match.group(1)
                    
                    # Skip if we already have this video
                    if video_id in [v['id'] for v in videos]:
                        continue
                    
                    # Try multiple selectors for video titles
                    title = None
                    selectors = [
                        '#video-title',
                        'yt-formatted-string#video-title',
                        '[id="video-title"]',
                        'h3 a',
                        'span[title]',
                        '.ytd-rich-grid-media #video-title'
                    ]
                    
                    for selector in selectors:
                        title_element = await element.query_selector(selector)
                        if title_element:
                            # Try different attributes for title
                            title = await title_element.get_attribute('title')
                            if not title:
                                title = await title_element.text_content()
                            if title:
                                break
                    
                    if not title:
                        # Try to get title from aria-label
                        aria_label = await element.get_attribute('aria-label')
                        if aria_label:
                            title = aria_label
                    
                    if title:
                        # Clean up title (remove duration info if present)
                        title = re.sub(r'\s*\d+\s+minutes?(?:,\s*\d+\s+seconds?)?\s*$', '', title.strip())
                        videos.append({'id': video_id, 'title': title})
                        processed_count += 1
                        if processed_count <= 3:  # Show debug for first few videos
                            print(f"DEBUG: Added video: {title} ({video_id})")
                        elif processed_count == 4:
                            print("DEBUG: Processing more videos...")
                    
                    # Stop after finding enough videos
                    if len(videos) >= 50:
                        break
        
        print(f"DEBUG: Total videos collected: {len(videos)}")
        await browser.close()
        return videos

async def main():
    videos = await scrape_youtube_trending()
    
    print(f"Found {len(videos)} trending videos:\n")
    for i, video in enumerate(videos, 1):
        print(f"{i:2d}. {video['title']}")
        print(f"    Video ID: {video['id']}")
        print(f"    URL: https://www.youtube.com/watch?v={video['id']}")
        print()

if __name__ == "__main__":
    asyncio.run(main())
