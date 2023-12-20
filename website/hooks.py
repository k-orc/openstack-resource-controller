import os

def on_post_build(config):
    dir = os.path.join(config.site_dir, "t/p")
    os.makedirs(dir, exist_ok=True)
    open(os.path.join(dir, "script.js"), 'w').close()
