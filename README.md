# LINUXUS

> Linuxus, a Docker-based service that provides Ubuntu shell environments via a web browser for Linux education

## 🌐 Actual Service

> ![LOGIN](./doc/fig/01-login.png)
> 

> ![SHELL](./doc/fig/02-in_service.png)
>

## 🚀 Usage

0. Clone repository:

   ```bash
   git clone https://github.com/elecbug/linuxus
   cd linuxus
   ```

1. If required packages (Go, Docker, etc.) are not installed, run the following:

   ```bash
   # Install Go package
   sudo snap install go --classic
   ```

   ```bash
   # Install Docker packages
   ./util/docker_reinstall.sh
   ```

2. Build the hash generator:

   ```bash
   ./util/make_hash/build.sh
   ```

   This will generate the following executable:

   ```bash
   ls util/make_hash.out
   ```

3. Add the authentication file:

   ```bash
   mkdir -p src/data
   touch src/data/AUTH_LIST
   ```

4. Create user accounts by appending credentials using:

   ```bash
   ./util/make_hash.out <ID> <PASSWORD> >> src/data/AUTH_LIST
   ```

   You can change the **ADMIN account ID** by modifying the `ADMIN_USER_ID` value in the `src/config.env` file.
   The default value is `alpha`.

   ```bash
   ...
   ADMIN_USER_ID=alpha
   ...
   ```

5. Start the services (containers) for each user:

   ```bash
   ./util/simple_build_and_run.sh <OPTION>
   ```

   **⚙️ Options**

    You can pass the following options to control container behavior:

    * **`--clear`**
    Reset all user directories (volumes).

    * **`--restart`**
    Build compose and restart all user containers.

    * **`--down`**
    Stop all user containers.

    * **`--up`**
    Build compose and start all user containers. (`--down` not included)

6. After running the service, a `src/volumes` directory will be created automatically.

   Inside this directory:

   * User directories are located under the `homes` folder.
   * A shared directory (`share`) will be created.
   * A read-only directory (`readonly`) will be created.

   **Directory Structure**

   ```
   src/volumes/
   ├── homes/
   │   ├── <USER1>/
   │   ├── <USER2>/
   │   └── ...
   ├── share/
   └── readonly/
   ```

   **Directory Permissions**

   * **User directories (`homes/<USER>`)**

     * Accessible only by the corresponding user.
     * Mounted to `/home/<USER>` inside each container.

   * **`share` directory**

     * Accessible by all users.
     * Read, write, and execute permissions are allowed.
     * Mounted to `/home/share` inside each container.

   * **`readonly` directory**

     * Read and execute permissions for all users.
     * **Write access is restricted to the admin account only**.
     * Mounted to `/home/readonly` inside each container.

## 📄 License

This project is licensed under the [**MIT License**](./LICENSE).

## 🌱 Open Source & Contributions

This project is open source, and contributions are always welcome.

* Feel free to open issues for bugs, questions, or suggestions.
* Pull requests (PRs) are highly encouraged.
* Any form of contribution — code, documentation, or ideas — is appreciated.

## 🚧 Upcoming Features

We are currently working on adding a **sign-up feature that can be used during runtime**.

This feature is under development and will be available in a future update.
