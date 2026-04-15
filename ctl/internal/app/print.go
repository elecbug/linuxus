package app

import "fmt"

func (a *App) PrintSummary() {
	fmt.Println("Runtime service plan prepared.")
	fmt.Println()
	fmt.Println("Config file:")
	fmt.Printf("  %s\n\n", a.configFile)

	fmt.Println("Login URL:")
	fmt.Printf("  http://localhost:%d/%s\n\n", a.Config.AuthService.Container.ExternalPort, a.Config.AuthService.URLPath.Login)

	fmt.Println("Images:")
	fmt.Printf("  AUTH=%s\n", a.authImageName())
	fmt.Printf("  USER=%s\n", a.userImageName())
	fmt.Printf("  MANAGER=%s\n\n", a.managerImageName())

	fmt.Println("Always-on services:")
	fmt.Printf("  AUTH=%s\n", a.Config.AuthService.Container.Name)
	fmt.Printf("  MANAGER=%s\n", a.Config.ManagerService.Container.Name)
	fmt.Printf("  MANAGER_NET=%s\n\n", a.Config.ManagerService.Container.Network)

	fmt.Println("On-demand user runtime:")
	fmt.Printf("  CONTAINER PREFIX=%s\n", a.Config.UserService.Container.NamePrefix)
	fmt.Printf("  NETWORK PREFIX=%s\n", a.Config.UserService.Container.NetworkPrefix)

	fmt.Println()
	fmt.Println("Run:")
	fmt.Printf("  %s -u\n", a.execPath)
}
