#import <Cocoa/Cocoa.h>
#import <WebKit/WebKit.h>
#import <IOKit/IOKitLib.h>
#import <IOKit/graphics/IOGraphicsLib.h>
#include <stdio.h>

extern void guiReadyCallback();
extern void guiWindowCloseCallback(int);

typedef struct ScreenList ScreenList;
struct ScreenList {
	int left, top;
	int width, height;
	int usableLeft, usableTop;
	int usableWidth, usableHeight;
	int isPrimary;
	ScreenList* next;
};

//
//	Map of Window handle -> NSWindows. 
//

NSMutableDictionary* windowLookup;

//
//	Browser Delegate: Catches load notifications, and sets the window title based on document title.
//

@interface BrowserDelegate : WebView {
}
@end

@implementation BrowserDelegate
- (void)webView:(WebView *)sender didFinishLoadForFrame:(WebFrame *)frame
{
	//	Get the HTML page title and use it as the window title.
	NSString *title = [sender stringByEvaluatingJavaScriptFromString:@"document.title"];
	[[sender window] setTitle: title];
}
@end
BrowserDelegate* browserDelegate;

//
//	Window Delegate:	Catches window events.
//

@interface WindowDelegate : NSObject <NSWindowDelegate> {
}
@end

@implementation WindowDelegate
-(void)windowWillClose:(NSNotification*)notification {
	NSWindow* window = (NSWindow*)[notification object];
	NSArray *windowIds = [windowLookup allKeysForObject: window];
	if ([windowIds count] > 0) {
		int windowId = [[windowIds objectAtIndex:0] intValue];
		guiWindowCloseCallback(windowId);
	}
	
	[[NSApplication sharedApplication] stopModal];
}
@end
WindowDelegate* windowDelegate;

//
//	App Delegate: Catches app ready notification, informs Go program that the GUI is ready to rock. Opens new Windows.
//

@interface AppDelegate : NSObject <NSApplicationDelegate> {
}
@end

@implementation AppDelegate
-(void)applicationDidFinishLaunching:(NSNotification *)aNotification {
	guiReadyCallback();
}

-(void)openWindow:(const char*)url withId:(int)id withFlags:(int)flags asModal:(bool)modal {
	NSString* nsurl = [NSString stringWithCString:url encoding:NSUTF8StringEncoding];

	//	Create a window.
	NSWindow* window = [[NSWindow alloc] 
		initWithContentRect: NSMakeRect(0, 0, 500, 400)
		styleMask: flags
		backing:NSBackingStoreBuffered
		defer:NO
	];
	[window makeKeyAndOrderFront:nil];
	[window setDelegate:windowDelegate];
	
	if (modal)
		[window setLevel:NSModalPanelWindowLevel];

	//	Determine an autosave name based on the URL.
	NSError* error = nil;
	NSString* basename = [nsurl lastPathComponent];
	NSRegularExpression* regex = [NSRegularExpression regularExpressionWithPattern:@"[^a-z0-9]" options:NSRegularExpressionCaseInsensitive error:&error];
	NSString* autosaveName = [regex stringByReplacingMatchesInString:basename options:0 range:NSMakeRange(0, [basename length]) withTemplate:@""];
	[window center];
	[window setFrameAutosaveName:autosaveName];

	//	Create a WebView, and load the specified URL.
	NSView *superview = [window contentView];
	NSRect frame = NSMakeRect(0, 0, 500, 400);
	WebView* webView = [[WebView alloc] initWithFrame:frame];
	[superview addSubview:webView]; 
	[webView setMainFrameURL:nsurl];
	[webView setFrameLoadDelegate: browserDelegate];

	//	Constrain the WebView to 100% fill the surrounding window.
	[webView setTranslatesAutoresizingMaskIntoConstraints:NO];
	NSDictionary *viewBindings = NSDictionaryOfVariableBindings(webView);
	[superview addConstraints:[NSLayoutConstraint constraintsWithVisualFormat:@"H:|[webView]|" options:0 metrics:nil views:viewBindings]];
	[superview addConstraints:[NSLayoutConstraint constraintsWithVisualFormat:@"V:|[webView]|" options:0 metrics:nil views:viewBindings]];
	
	//	Store a reference to this window for later lookups.
	windowLookup[@(id)] = window;
	NSString* wkey = [NSString stringWithFormat:@"w%p", window];
	windowLookup[wkey] = @(id);
}

-(void)closeWindow:(NSWindow*)window {
	[window close];
}

@end
AppDelegate* appDelegate;

//
//	OpenWindow; exported method, callable from Go code. Calls App Delegate's open window function in the main NSApp thread. Returns an int window handle.
//

void OpenWindow(int windowId, const char* url, int flags, bool modal) {
	dispatch_sync(dispatch_get_main_queue(), ^{
		[appDelegate openWindow:url withId:windowId withFlags:flags asModal:modal];
	});
}

//
//	CloseWindow; exported method, closes the specified window by id.
//

void CloseWindow(int id) {
	NSWindow* window = (NSWindow*)[windowLookup objectForKey:@(id)];
	if (window != nil) {
		[appDelegate performSelectorOnMainThread:@selector(closeWindow:) withObject:window waitUntilDone:YES];
		[windowLookup removeObjectForKey:@(id)];
	}
}

//
//	StartApp; exported method, prepares an NSApp to manage all Cocoa windows. Never returns.
//

int StartApp() {
	[NSAutoreleasePool new];
	[NSApplication sharedApplication];
	[NSApp setActivationPolicy:NSApplicationActivationPolicyRegular];
	
	appDelegate = [[AppDelegate new] autorelease];
	[NSApp setDelegate: appDelegate]; 
	
	browserDelegate = [[BrowserDelegate new] autorelease];
	windowDelegate = [[WindowDelegate new] autorelease];
	windowLookup = [NSMutableDictionary dictionary];

	[[NSUserDefaults standardUserDefaults] registerDefaults:[NSDictionary dictionaryWithObjectsAndKeys:
		[NSNumber numberWithBool:YES], @"WebKitDeveloperExtras",
		[NSNumber numberWithBool:YES], @"WebKitScriptDebuggerEnabled",
		nil
	]];
	
	//	Create a menu bar.
	id menubar = [[NSMenu new] autorelease];
	id appMenuItem = [[NSMenuItem new] autorelease];
	[menubar addItem:appMenuItem];
	[NSApp setMainMenu:menubar];
	
	id appMenu = [[NSMenu new] autorelease];
	id appName = [[NSProcessInfo processInfo] processName];
	id quitTitle = [@"Quit " stringByAppendingString:appName];
	id quitMenuItem = [[[NSMenuItem alloc]
		initWithTitle:quitTitle
	    	action:@selector(terminate:) keyEquivalent:@"q"]
		autorelease];
	[appMenu addItem:quitMenuItem];
	[appMenuItem setSubmenu:appMenu];
	
	[NSApp activateIgnoringOtherApps:YES];
	[NSApp run];
	return 0;
}

//
//	StopApp; exported method, used to stop running the nsapp.
//

void StopApp() {
	[NSApp terminate:0];
}

void SetWindowTitle(int id, const char* title) {
	NSWindow* window = (NSWindow*)[windowLookup objectForKey:@(id)];
	if (window != nil) {
		dispatch_async(dispatch_get_main_queue(), ^{
			[window setTitle:[NSString stringWithCString:title encoding:NSUTF8StringEncoding]];
		});
	}
}

void SetWindowSize(int id, int width, int height) {
	NSWindow* window = (NSWindow*)[windowLookup objectForKey:@(id)];
	if (window != nil) {
		dispatch_async(dispatch_get_main_queue(), ^{
			NSRect frame = [window frame];
			frame.size.width = width;
			frame.size.height = height;
			[window setFrame:frame display:YES animate:NO];
		});
	}
}

void SetWindowPosition(int id, int left, int top) {
	NSWindow* window = (NSWindow*)[windowLookup objectForKey:@(id)];
	if (window != nil) {
		dispatch_async(dispatch_get_main_queue(), ^{
			//	Cocoa is weird. Coords are upside-down. Compensate for it.
			NSScreen* mainScreen = [NSScreen mainScreen];
			
			NSRect frame = [window frame];
			frame.origin.x = left;
			frame.origin.y = mainScreen.visibleFrame.size.height - top - frame.size.height;
			[window setFrame:frame display:YES animate:NO];
		});
	}
}

void GetScreenSize(int* width, int* height) {
	NSScreen* main = [NSScreen mainScreen];
	*width = main.visibleFrame.size.width;
	*height = main.visibleFrame.size.height;
}

void RememberWindowGeometry(int id, const char* key) {
	NSWindow* window = (NSWindow*)[windowLookup objectForKey:@(id)];
	if (window != nil) {
		dispatch_async(dispatch_get_main_queue(), ^{
			[window setFrameAutosaveName:[NSString stringWithCString:key encoding:NSUTF8StringEncoding]];
		});
	}
}

void RunModal(int id) {
	NSWindow* window = (NSWindow*)[windowLookup objectForKey:@(id)];
	if (window != nil) {
		dispatch_async(dispatch_get_main_queue(), ^{
			[NSApp runModalForWindow: window];
		});		
	}
}
