package app

import "fmt"

func (a *App) PrintSummary() {
	adminSafe := sanitizeName(a.Config.Admin.UserID)

	fmt.Printf("Generated %s\n\n", a.Config.Compose.OutputFile)
	fmt.Println("Config file:")
	fmt.Printf("  %s\n\n", a.ConfigFile)

	fmt.Println("Login URL:")
	fmt.Printf("  http://localhost:%d/%s\n\n", a.Config.AuthService.ExternalPort, a.Config.URLPaths.Login)

	fmt.Println("Users:")
	for i := range a.UserIDs {
		fmt.Printf("  ID=%s SERVICE=%s NET=%s\n",
			a.UserIDs[i],
			a.Config.UserService.ContainerNamePrefix+a.SafeIDs[i],
			a.Config.UserService.NetworkPrefix+a.SafeIDs[i],
		)
	}
	fmt.Printf("  ADMIN=%s NET=%s\n\n",
		a.Config.Admin.UserID,
		a.Config.UserService.NetworkPrefix+adminSafe,
	)

	fmt.Println("Run:")
	fmt.Printf("  sudo docker compose -f %s up -d --build\n", a.Config.Compose.OutputFile)
}
