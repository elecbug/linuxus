# 🚀 Usage

## 0. Clone Repository

```bash
git clone https://github.com/elecbug/linuxus
cd linuxus
```

---

## 1. Install Dependencies

### Go

```bash
sudo snap install go --classic
```

### Docker

```bash
sudo apt install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
```

---

## 2. Build Controller

Build the control CLI:

```bash
./ctl/build.sh
```

Generated executable:

```bash
./linuxusctl --help
```

---

## 3. Run Service

### 3.1 Start / Manage Services

```bash
./linuxusctl <OPTION>
```

### ⚙️ Available Options

| Command                           | Description                                                                                      |
| --------------------------------- | ------------------------------------------------------------------------------------------------ |
| `help`                            | Show help message                                                                                |
| `up`                              | Build images and start services                                                                  |
| `down`                            | Stop and remove services                                                                         |
| `restart`                         | Restart services                                                                                 |
| `ps [all\|container\|network]`    | Show status about linuxus service                                                                |
| `add-user <USERNAME>`             | Add a new user                                                                                   |
| `remove-user <USERNAME>`          | Remove an existing user                                                                          |
| `volume-clean <OPTION\|USERNAME>` | Remove all user directories if the option is all, otherwise remove specific user directory       |
| `ensure-disk <OPTION\|USERNAME>`  | Create a missing user directory if the option is all, otherwise create a specific user directory |

---

### 3.2 Example Usage

```bash
./linuxusctl up                    # Build and start
./linuxusctl restart               # Restart
./linuxusctl ps network            # Show network status of linuxus service
```

---

## 4. Setup Authentication (Signup-based)

> [!NOTE]
> Linuxus provides a built-in signup system via the web interface.

---

### 4.1 Enable Signup

Edit `cfg/config.yml`:

```yml
auth_service:
  allow_signup: true
```

A **Signup** link will appear on the login page.

---

### 4.2 User Registration Flow

1. User opens the login page
2. Clicks **Signup**
3. Enters ID and password
4. Account is registered

> [!IMPORTANT]
> Newly registered users are **not immediately usable**

---

### 4.3 Activate User Environment

After signup, the host must initialize user environments:

```bash
./linuxusctl -e
```

or:

```bash
./linuxusctl ensure-disk
```

This step:

* Creates home directories
* Mounts volumes
* Activates user accounts

---

### 4.4 Admin Account

* Default admin ID: `alpha`
* Configurable in:

```yml
manager_service:
  admin_id: alpha
```

---

## 5. Volume Structure

```
volumes/
├── homes/
├── share/
└── readonly/
```

---

## 6. Directory Permissions

### 👤 User (`homes/<USER>`)

* Private
* Mounted to `/home/<USER>`

### 📂 Share

* RWX for all users
* `/home/share`

### 🔒 Readonly

* Read/execute for users
* Write for admin only
* `/home/readonly`

---

## 🌐 APPENDIX - Preview Image

> ![](./fig/04-arch.png)
> Linuxus Architecture Diagram

> ![](./fig/01-login.png)
> Login Page

> ![](./fig/02-shell_1.png)
> Shell Page - Access

> ![](./fig/03-shell_2.png)
> Shell Page - Test GCC

---

## 📎 APPENDIX - Manual User Setup (Legacy)

### Build Hash Generator

```bash
./util/make_hash/build.sh
```

### Create Authentication File

```bash
mkdir -p data
touch data/AUTH_LIST
```

### Add Users

```bash
./util/make_hash.out <ID> <PASSWORD> >> data/AUTH_LIST
```

Recommended for:

* Initial admin bootstrap
* Bulk provisioning
* Signup-disabled environments