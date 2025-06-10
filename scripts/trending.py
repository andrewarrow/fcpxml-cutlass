#!/usr/bin/env python3
import asyncio
import re
from playwright.async_api import async_playwright

async def fetch_channel_description(page, channel_name):
    """Fetch channel description by searching for the channel"""
    if not channel_name:
        return None
    
    try:
        # Navigate to channel search
        search_url = f"https://www.youtube.com/results?search_query={channel_name}&sp=EgIQAg%253D%253D"
        await page.goto(search_url)
        await page.wait_for_load_state('networkidle')
        
        # Click on the first channel result
        channel_link = await page.query_selector('a[href*="/channel/"], a[href*="/@"]')
        if channel_link:
            await channel_link.click()
            await page.wait_for_load_state('networkidle')
            
            # Look for channel description
            description_selectors = [
                '#description yt-formatted-string',
                '#description-container yt-formatted-string',
                '.ytd-channel-about-metadata-renderer yt-formatted-string',
                '#about-description yt-formatted-string'
            ]
            
            for selector in description_selectors:
                desc_element = await page.query_selector(selector)
                if desc_element:
                    description = await desc_element.text_content()
                    if description and description.strip():
                        return description.strip()
        
        return None
    except Exception as e:
        print(f"DEBUG: Error fetching channel description for {channel_name}: {e}")
        return None

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
                        
                        # Try to find duration
                        duration = None
                        duration_selectors = [
                            '.ytd-thumbnail-overlay-time-status-renderer span',
                            '#overlays .ytd-thumbnail-overlay-time-status-renderer',
                            '.badge-shape-wiz__text',
                            'span.style-scope.ytd-thumbnail-overlay-time-status-renderer',
                            '[aria-label*="minutes"]'
                        ]
                        
                        # Look for duration in the parent container
                        parent = await element.evaluate('el => el.closest("ytd-rich-grid-media, ytd-video-renderer, ytd-compact-video-renderer")')
                        if parent:
                            for duration_selector in duration_selectors:
                                duration_element = await page.query_selector(f'ytd-rich-grid-media:has(a[href*="{video_id}"]) {duration_selector}')
                                if not duration_element:
                                    duration_element = await page.query_selector(f'ytd-video-renderer:has(a[href*="{video_id}"]) {duration_selector}')
                                if duration_element:
                                    duration = await duration_element.text_content()
                                    if duration and duration.strip():
                                        duration = duration.strip()
                                        break
                        
                        # Try to find channel name
                        channel_name = None
                        channel_selectors = [
                            '#channel-name a',
                            '.ytd-channel-name a',
                            'a[href*="/channel/"] #text',
                            'a[href*="/@"] #text',
                            '.yt-simple-endpoint.style-scope.yt-formatted-string',
                            '#owner-text a'
                        ]
                        
                        # Look for channel name in the video container
                        for channel_selector in channel_selectors:
                            channel_element = await page.query_selector(f'ytd-rich-grid-media:has(a[href*="{video_id}"]) {channel_selector}')
                            if not channel_element:
                                channel_element = await page.query_selector(f'ytd-video-renderer:has(a[href*="{video_id}"]) {channel_selector}')
                            if channel_element:
                                channel_name = await channel_element.text_content()
                                if channel_name and channel_name.strip():
                                    channel_name = channel_name.strip()
                                    break
                        
                        # Try to find video description (snippet)
                        video_description = None
                        description_selectors = [
                            '#description-text',
                            '.ytd-video-meta-block #description-text',
                            'yt-formatted-string#description-text',
                            '.metadata-snippet-text'
                        ]
                        
                        for desc_selector in description_selectors:
                            desc_element = await page.query_selector(f'ytd-rich-grid-media:has(a[href*="{video_id}"]) {desc_selector}')
                            if not desc_element:
                                desc_element = await page.query_selector(f'ytd-video-renderer:has(a[href*="{video_id}"]) {desc_selector}')
                            if desc_element:
                                video_description = await desc_element.text_content()
                                if video_description and video_description.strip():
                                    video_description = video_description.strip()
                                    break
                        
                        videos.append({
                            'id': video_id, 
                            'title': title, 
                            'duration': duration,
                            'channel_name': channel_name,
                            'video_description': video_description,
                            'channel_description': None  # Will be fetched separately if needed
                        })
                        processed_count += 1
                        if processed_count <= 3:  # Show debug for first few videos
                            print(f"DEBUG: Added video: {title} ({video_id}) - Duration: {duration} - Channel: {channel_name}")
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
        duration_text = f" ({video['duration']})" if video['duration'] else " (duration unknown)"
        print(f"    Duration:{duration_text}")
        
        if video['channel_name']:
            print(f"    Channel: {video['channel_name']}")
        else:
            print("    Channel: (unknown)")
        
        if video['video_description']:
            # Truncate description if too long
            desc = video['video_description']
            if len(desc) > 100:
                desc = desc[:100] + "..."
            print(f"    Description: {desc}")
        else:
            print("    Description: (none available)")
        
        print(f"    URL: https://www.youtube.com/watch?v={video['id']}")
        print()

if __name__ == "__main__":
    asyncio.run(main())
