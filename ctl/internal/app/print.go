package app

import "fmt"

func (a *App) PrintSummary() {
	adminSafe := sanitizeName(a.Config.UserService.Container.Admin.UserID)

	fmt.Println("Runtime service plan prepared.")
	fmt.Println()
	fmt.Println("Config file:")
	fmt.Printf("  %s\n\n", a.ConfigFile)

	fmt.Println("Login URL:")
	fmt.Printf("  http://localhost:%d/%s\n\n", a.Config.AuthService.Container.ExternalPort, a.Config.AuthService.URLPath.Login)

	fmt.Println("Images:")
	fmt.Printf("  AUTH=%s\n", a.authImageName())
	fmt.Printf("  USER=%s\n\n", a.userImageName())

	fmt.Println("Users:")
	for i := range a.UserIDs {
		fmt.Printf("  ID=%s CONTAINER=%s NET=%s\n",
			a.UserIDs[i],
			a.Config.UserService.Container.NamePrefix+a.SafeIDs[i],
			a.Config.UserService.Container.NetworkPrefix+a.SafeIDs[i],
		)
	}
	fmt.Printf("  ADMIN=%s NET=%s\n\n",
		a.Config.UserService.Container.Admin.UserID,
		a.Config.UserService.Container.NetworkPrefix+adminSafe,
	)

	fmt.Println("Run:")
	fmt.Printf("  %s -u\n", a.ExecPath)
}
