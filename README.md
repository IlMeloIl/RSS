# Agregador de RSS

Um agregador de feeds RSS de linha de comando construído em Go. Esta aplicação permite que você se inscreva em feeds RSS, gerencie suas inscrições e navegue por publicações de seus sites favoritos diretamente do terminal.

## Funcionalidades

- Registro e autenticação de usuários
- Adicionar e gerenciar feeds RSS
- Seguir/deixar de seguir feeds
- Navegar por publicações dos feeds que você segue
- Agregação automática de feeds com intervalos personalizáveis

## Pré-requisitos

- Go 1.24 ou superior
- Banco de dados PostgreSQL

## Instalação

1. Clone o repositório:
   ```
   git clone https://github.com/IlMeloIl/RSS.git
   cd RSS
   ```

2. Instale as dependências:
   ```
   go mod download
   ```

3. Configure o banco de dados:
   - Crie um banco de dados PostgreSQL
   - Aplique as migrações de esquema em ordem a partir do diretório `sql/schema`
   - Você pode usar o [goose](https://github.com/pressly/goose) para aplicar as migrações:
     ```
     goose -dir sql/schema postgres "sua-string-de-conexão" up
     ```

4. Configure a aplicação:
   - Crie um arquivo `.gatorconfig.json` em seu diretório home com a seguinte estrutura:
     ```json
     {
       "db_url": "postgres://usuario:senha@localhost:5432/rss_db",
       "current_user_name": ""
     }
     ```
   - Substitua a string de conexão do banco de dados com seus detalhes reais do PostgreSQL

## Uso

### Gerenciamento de Usuários

Registrar um novo usuário:
```
go run . register <nome-de-usuário>
```

Fazer login como um usuário existente:
```
go run . login <nome-de-usuário>
```

Visualizar todos os usuários:
```
go run . users
```

### Gerenciamento de Feeds

Adicionar um novo feed:
```
go run . addfeed <nome> <url>
```

Listar todos os feeds disponíveis:
```
go run . feeds
```

Seguir um feed:
```
go run . follow <url>
```

Deixar de seguir um feed:
```
go run . unfollow <url>
```

Listar feeds que você está seguindo:
```
go run . following
```

### Leitura de Publicações

Navegar por publicações de feeds que você segue:
```
go run . browse [limite]
```
O parâmetro opcional `limite` controla quantas publicações exibir (padrão: 2)

### Agregação de Feeds

Iniciar o agregador de feeds para buscar novas publicações:
```
go run . agg <intervalo-de-tempo>
```
Onde `intervalo-de-tempo` está no formato de duração do Go (ex: `5s` para 5 segundos, `1m` para 1 minuto, `1h` para 1 hora)

### Outros Comandos

Resetar o banco de dados (remove todos os usuários e seus dados):
```
go run . reset
```

## Estrutura do Projeto

- `sql/schema/`: Arquivos de migração do esquema do banco de dados
- `sql/queries/`: Arquivos de consulta SQL usados pelo sqlc
- `internal/database/`: Código e modelos de banco de dados gerados
- `internal/config/`: Gerenciamento de configuração
- `handlers.go`: Manipuladores de comandos para todas as funcionalidades da aplicação
- `main.go`: Ponto de entrada da aplicação
- `middleware.go`: Middleware de autenticação
- `types&methods.go`: Estruturas de dados e métodos

## Como Funciona

1. A aplicação lê a configuração de `.gatorconfig.json` no seu diretório home
2. Comandos são processados através da interface de linha de comando
3. A maioria dos comandos requer um usuário logado (armazenado no arquivo de configuração)
4. Feeds RSS são buscados, analisados e armazenados no banco de dados
5. O agregador atualiza periodicamente os feeds para obter novas publicações