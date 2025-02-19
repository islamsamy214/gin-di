# Gin DI Framework

A lightweight, Laravel-inspired structure implementation for the [Gin Framework](https://github.com/gin-gonic/gin). Simplify dependency management, enhance modularity, and build scalable Go web applications with ease.

---

## 🚀 Features
  
- **Middleware Integration**: Seamlessly inject services into Gin's context.
- **Extensible Design**: Add custom services and providers with ease.
- **Laravel-Inspired**: Bring Laravel's elegance to your Go applications.
- **Database**: Starting with pgsql primarily, you can replace it with any other client **NO ORM USED, NOR SQLC** just a pure pgsql client [Go pgsql client](https://github.com/lib/pq)
- **Logging**: Daily logging, alongside with the CLI logging
- 
---

## 📂 Project Structure

```plaintext
/app
  /http
  /grpc
  /console
  /models
  /services
  /providers
/config
/routes
/bin
/tests
main.go
README.md
LICENSE
.gitignore
.env
.env.example
```

---

## 🤝 Contributing

We welcome contributions from the community! Please follow these steps:  

1. Fork the repository.  
2. Create a new branch (`git checkout -b feature-name`).  
3. Commit your changes (`git commit -m 'Add new feature'`).  
4. Push the branch (`git push origin feature-name`).  
5. Open a Pull Request.  

For more details, see [CONTRIBUTING.md](CONTRIBUTING.md).

---

## 📜 License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

---

## ⭐ Support

If you like this project, please give it a star ⭐! Your support means a lot!
