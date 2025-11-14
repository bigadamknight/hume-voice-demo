"""Verify that the setup is correct and all imports work."""
import sys
import logging
from rich.console import Console

console = Console()
logger = logging.getLogger(__name__)


def verify_imports():
    """Verify all required modules can be imported."""
    errors = []
    
    dependencies = [
        ("click", "click"),
        ("rich", "rich"),
        ("python-dotenv", "dotenv"),
        ("hume SDK", "hume"),
        ("certifi", "certifi"),
        ("sounddevice", "sounddevice"),
    ]
    
    for name, module in dependencies:
        try:
            __import__(module)
            console.print(f"[green]✓[/green] {name} imported")
        except ImportError as e:
            errors.append(f"✗ {name}: {e}")
            console.print(f"[red]✗[/red] {name}: {e}")
    
    # Verify local modules
    local_modules = [
        ("config", "config"),
        ("database", "database"),
        ("evi_client", "evi_client"),
        ("audio_handler", "audio_handler"),
        ("conversation_manager", "conversation_manager"),
        ("cli", "cli"),
    ]
    
    for name, module in local_modules:
        try:
            __import__(module)
            console.print(f"[green]✓[/green] {name} module imported")
        except ImportError as e:
            errors.append(f"✗ {name}: {e}")
            console.print(f"[red]✗[/red] {name}: {e}")
    
    if errors:
        console.print("\n[red]Errors found:[/red]")
        for error in errors:
            console.print(f"  {error}")
        return False
    
    console.print("\n[green]✓ All imports successful![/green]")
    return True


def verify_env():
    """Verify environment configuration."""
    try:
        from config import get_config
        config = get_config()
        console.print("[green]✓[/green] Environment variables configured")
        console.print(f"[dim]  HUME_API_KEY: {'*' * 20}[/dim]")
        console.print(f"[dim]  HUME_CONFIG_ID: {config.hume_config_id[:8]}...[/dim]")
        console.print(f"[dim]  ALLOW_INTERRUPT: {config.allow_interrupt}[/dim]")
        return True
    except ValueError as e:
        console.print(f"[red]⚠[/red] Configuration error: {e}")
        return False
    except Exception as e:
        console.print(f"[red]⚠[/red] Error verifying config: {e}")
        return False


if __name__ == "__main__":
    console.print("[blue]Verifying setup...[/blue]\n")
    
    imports_ok = verify_imports()
    console.print()
    env_ok = verify_env()
    
    if imports_ok and env_ok:
        console.print("\n[green]✓ Setup verification complete![/green]")
        sys.exit(0)
    else:
        console.print("\n[red]✗ Setup verification failed. Please fix the issues above.[/red]")
        sys.exit(1)
