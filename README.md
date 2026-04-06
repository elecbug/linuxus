# LINUXUS

Linuxus, Docker-based service that provides Ubuntu shells through web browsers for Linux teachers

## 🚀 Usage

0. Clone repository:

   ```bash
   git clone https://github.com/elecbug/linuxus
   cd linuxus
   ```

1. If packages (Go-lang, Docker, and etc) are not installed, run like below:

   ```bash
   sudo snap install go --classic
   ```

   ```bash
   ./util/docker_reinstall.sh
   ```

2. Build the hash generator:

   ```bash
   ./util/make_hash/build.sh
   ```

   This will create:

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
   The default value is `alpha`:

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

    * **`--clear-volume`**
    Reset all user directories (volumes) and restart containers.

    * **`--restart`**
    Restart all user containers.

    * **`--only-down`**
    Stop all user containers.

    * **`--only-up`**
    Create and start all user containers.

6. After running the service, a `src/volumes` directory will be created automatically.

   Inside this directory:

   * User directories are located under the `homes` folder
   * A shared directory (`share`) will be created
   * A read-only directory (`readonly`) will be created

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

     * Accessible only by the corresponding user
     * Mount to the `/home/<USER>` directory within each service

   * **`share` directory**

     * Accessible by all users
     * Read, write, and execute permissions are allowed
     * Mount to the `/home/share` directory within each service

   * **`readonly` directory**

     * Read and execute permissions for all users
     * **Write access is restricted to the admin account only**
     * Mount to the `/home/readonly` directory within each service

## 📄 License

This project is licensed under the [**MIT License**](./LICENSE).

## 🌱 Open Source & Contributions

This project is open source, and contributions are always welcome.

* Feel free to open issues for bugs, questions, or suggestions
* Pull requests (PRs) are highly encouraged
* Any form of improvement —code, documentation, or ideas— is appreciated

## 🚧 Upcoming Features

We are currently working on adding a **sign-up feature that can be used during service operation (runtime)**.

This feature is under development and will be available in a future update.
