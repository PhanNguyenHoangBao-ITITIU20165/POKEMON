package main

import (
    "bufio"
    "encoding/json"
    "fmt"
    "log"
    "math/rand"
    "net"
    "net/http"
    "os"
    "strconv"
    "strings"
    "time"
)

type Pokemon struct {
    Name           string   `json:"name"`
    Type           []string `json:"type"`
    HP             int      `json:"hp"`
    BaseExp        int      `json:"base_exp"`
    Attack         int      `json:"attack"`
    Defense        int      `json:"defense"`
    Speed          int      `json:"speed"`
    SpecialAttack  int      `json:"special_attack"`
    SpecialDefense int      `json:"special_defense"`
}

func fetchPokemonData(id int) (*Pokemon, error) {
    resp, err := http.Get(fmt.Sprintf("https://pokeapi.co/api/v2/pokemon/%d/", id))
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var result map[string]interface{}
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }

    types := []string{}
    for _, t := range result["types"].([]interface{}) {
        typeName := t.(map[string]interface{})["type"].(map[string]interface{})["name"].(string)
        types = append(types, typeName)
    }

    stats := make(map[string]int)
    for _, s := range result["stats"].([]interface{}) {
        statName := s.(map[string]interface{})["stat"].(map[string]interface{})["name"].(string)
        statValue := int(s.(map[string]interface{})["base_stat"].(float64))
        stats[statName] = statValue
    }

    pokemon := &Pokemon{
        Name:           result["name"].(string),
        Type:           types,
        BaseExp:        int(result["base_experience"].(float64)),
        HP:             stats["hp"],
        Attack:         stats["attack"],
        Defense:        stats["defense"],
        Speed:          stats["speed"],
        SpecialAttack:  stats["special-attack"],
        SpecialDefense: stats["special-defense"],
    }

    return pokemon, nil
}

func FetchAllPokemonData() {
    var pokemons []Pokemon

    for i := 1; i <= 100; i++ { // Fetch first 100 Pokémon
        pokemon, err := fetchPokemonData(i)
        if err != nil {
            log.Printf("Error fetching data for Pokémon ID %d: %v", i, err)
            continue
        }
        pokemons = append(pokemons, *pokemon)
    }

    jsonFile, err := os.Create("pokedex.json")
    if err != nil {
        log.Fatal(err)
    }
    defer jsonFile.Close()

    encoder := json.NewEncoder(jsonFile)
    encoder.SetIndent("", "  ")
    err = encoder.Encode(pokemons)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("Pokédex successfully created!")
}

func calculateNormalDamage(attacker *Pokemon, defender *Pokemon) int {
    if attacker == nil || defender == nil {
        log.Println("Error in calculateNormalDamage: attacker or defender is nil")
        return 0
    }
    damage := attacker.Attack - defender.Defense
    if damage < 0 {
        damage = 0
    }
    return damage
}

func calculateTypeEffectiveness(attackerType, defenderType string) float64 {
    if defenderTypes, ok := typeChart[attackerType]; ok {
        if multiplier, exists := defenderTypes[defenderType]; exists {
            return multiplier
        }
    }
    return 1.0 // Default effectiveness (no effect)
}

func calculateSpecialDamage(attacker, defender *Pokemon) int {
    maxMultiplier := 1.0
    for _, attackerType := range attacker.Type {
        for _, defenderType := range defender.Type {
            multiplier := calculateTypeEffectiveness(attackerType, defenderType)
            if multiplier > maxMultiplier {
                maxMultiplier = multiplier
            }
        }
    }
    damage := int(float64(attacker.SpecialAttack)*maxMultiplier - float64(defender.SpecialDefense))
    if damage < 0 {
        damage = 0
    }
    return damage
}


type Client struct {
    conn          net.Conn
    team          []*Pokemon
    isActive      bool
    activePokemon *Pokemon
    reader        *bufio.Reader
}

type TypeEffectiveness struct {
    AttackingType string  `json:"attacking_type"`
    DefendingType string  `json:"defending_type"`
    Multiplier    float64 `json:"multiplier"`
}

// Type chart as a nested map for faster lookup
var typeChart = map[string]map[string]float64{
    "Normal": {
        "Rock": 0.5, "Ghost": 0.0, "Steel": 0.5,
    },
    "Fire": {
        "Fire": 0.5, "Water": 0.5, "Grass": 2.0, "Ice": 2.0,
        "Bug": 2.0, "Rock": 0.5, "Dragon": 0.5, "Steel": 2.0,
    },
    "Water": {
        "Fire": 2.0, "Water": 0.5, "Grass": 0.5, "Ground": 2.0,
        "Rock": 2.0, "Dragon": 0.5,
    },
    "Electric": {
        "Water": 2.0, "Electric": 0.5, "Grass": 0.5, "Ground": 0.0,
        "Flying": 2.0, "Dragon": 0.5,
    },
    "Grass": {
        "Fire": 0.5, "Water": 2.0, "Grass": 0.5, "Poison": 0.5,
        "Ground": 2.0, "Flying": 0.5, "Bug": 0.5, "Rock": 2.0,
        "Dragon": 0.5, "Steel": 0.5,
    },
    "Ice": {
        "Fire": 0.5, "Water": 0.5, "Grass": 2.0, "Ice": 0.5,
        "Ground": 2.0, "Flying": 2.0, "Dragon": 2.0, "Steel": 0.5,
    },
    "Fighting": {
        "Normal": 2.0, "Ice": 2.0, "Poison": 0.5, "Flying": 0.5,
        "Psychic": 0.5, "Bug": 0.5, "Rock": 2.0, "Ghost": 0.0,
        "Dark": 2.0, "Steel": 2.0, "Fairy": 0.5,
    },
    "Poison": {
        "Grass": 2.0, "Poison": 0.5, "Ground": 0.5, "Rock": 0.5,
        "Ghost": 0.5, "Steel": 0.0, "Fairy": 2.0,
    },
    "Ground": {
        "Fire": 2.0, "Electric": 2.0, "Grass": 0.5, "Poison": 2.0,
        "Flying": 0.0, "Bug": 0.5, "Rock": 2.0, "Steel": 2.0,
    },
    "Flying": {
        "Electric": 0.5, "Grass": 2.0, "Fighting": 2.0, "Bug": 2.0,
        "Rock": 0.5, "Steel": 0.5,
    },
    "Psychic": {
        "Fighting": 2.0, "Poison": 2.0, "Psychic": 0.5, "Dark": 0.0,
        "Steel": 0.5,
    },
    "Bug": {
        "Fire": 0.5, "Grass": 2.0, "Fighting": 0.5, "Poison": 0.5,
        "Flying": 0.5, "Psychic": 2.0, "Ghost": 0.5, "Dark": 2.0,
        "Steel": 0.5, "Fairy": 0.5,
    },
    "Rock": {
        "Fire": 2.0, "Ice": 2.0, "Fighting": 0.5, "Ground": 0.5,
        "Flying": 2.0, "Bug": 2.0, "Steel": 0.5,
    },
    "Ghost": {
        "Normal": 0.0, "Psychic": 2.0, "Ghost": 2.0, "Dark": 0.5,
    },
    "Dragon": {
        "Dragon": 2.0, "Steel": 0.5, "Fairy": 0.0,
    },
    "Dark": {
        "Fighting": 0.5, "Psychic": 2.0, "Ghost": 2.0, "Dark": 0.5,
        "Fairy": 0.5,
    },
    "Steel": {
        "Fire": 0.5, "Water": 0.5, "Electric": 0.5, "Ice": 2.0,
        "Rock": 2.0, "Steel": 0.5, "Fairy": 2.0,
    },
    "Fairy": {
        "Fire": 0.5, "Poison": 0.5, "Fighting": 2.0, "Dragon": 2.0,
        "Dark": 2.0, "Steel": 0.5,
    },
}


func getNextActivePokemon(client *Client) *Pokemon {
    for _, p := range client.team {
        if p.HP > 0 {
            return p
        }
    }
    return nil
}

func handleConnection(conn net.Conn, pokedex []Pokemon, clients map[net.Conn]*Client, done chan struct{}) {
    defer conn.Close()

    client := &Client{conn: conn, isActive: false, reader: bufio.NewReader(conn)}
    clients[conn] = client

    fmt.Fprintln(conn, "Welcome to the Pokémon battle!")
    fmt.Fprintln(conn, "Choose 3 Pokémon for your team:")
    for i, p := range pokedex {
        fmt.Fprintf(conn, "%d. %s (%v)\n", i+1, p.Name, p.Type)
    }
    for i := 0; i < 3; i++ {
        fmt.Fprint(conn, "Enter number for Pokémon: ")
        input, err := client.reader.ReadString('\n')
        if err != nil {
            fmt.Println("Error reading from client:", err)
            return
        }
        input = strings.TrimSpace(input)

        choice, err := strconv.Atoi(input)
        if err != nil || choice < 1 || choice > len(pokedex) {
            fmt.Fprintln(conn, "Invalid choice. Try again.")
            i--
            continue
        }

        client.team = append(client.team, &pokedex[choice-1])
    }

    done <- struct{}{}

    for {
        if client.isActive && len(client.team) > 0 {
            reader := bufio.NewReader(conn)
            activePokemon := client.team[0]

            fmt.Fprintf(conn, "Available attacks for %s:\n", activePokemon.Name)
            fmt.Fprintln(conn, "1. Normal Attack")
            fmt.Fprintln(conn, "2. Special Attack")

            fmt.Fprintln(conn, "Enter attack number:")
            attackInput, _ := reader.ReadString('\n')
            attackInput = strings.TrimSpace(attackInput)
            attackChoice, err := strconv.Atoi(attackInput)
            if err != nil || attackChoice < 1 || attackChoice > 2 {
                fmt.Fprintln(conn, "Invalid attack choice. Try again.")
                continue
            }

            attacker := client.team[0]

            // Find opponent client
            var opponentClient *Client
            for otherConn, otherClient := range clients {
                if otherConn != conn {
                    opponentClient = otherClient
                    break
                }
            }

            if opponentClient == nil || len(opponentClient.team) == 0 {
                fmt.Fprintln(conn, "Opponent has no Pokémon left!")
                continue
            }

            // Select the opponent's active Pokémon as defender
            defender := opponentClient.team[0]

            var damage int
            switch attackChoice {
            case 1:
                damage = calculateNormalDamage(attacker, defender)
            case 2:
                damage = calculateSpecialDamage(attacker, defender)
            }

            fmt.Fprintf(conn, "%s attacks %s for %d damage!\n", attacker.Name, defender.Name, damage)
            defender.HP -= damage

            // Fainting and Winner Determination
            if defender.HP <= 0 {
                fmt.Fprintf(conn, "%s fainted!\n", defender.Name)
                fmt.Fprintf(opponentClient.conn, "%s fainted!\n", defender.Name)

                // Remove fainted Pokémon from the opponent's team
                opponentClient.team = append(opponentClient.team[:0], opponentClient.team[1:]...)

                if len(opponentClient.team) == 0 {
                    // Announce the winner
                    fmt.Fprintln(conn, "All opponent's Pokémon have fainted. You win!")
                    fmt.Fprintln(opponentClient.conn, "All your Pokémon have fainted. You lose!")
                    break // Exit the loop
                }
            }

            // Switch Turns
            client.isActive = false
            opponentClient.isActive = true
            fmt.Fprintln(opponentClient.conn, "Your turn!")
        } else {
            // Wait for your turn
            fmt.Fprintln(conn, "Waiting for your turn...")
            time.Sleep(3 * time.Second) // Adjust wait time as needed
        }
    }

    for otherConn, otherClient := range clients {
        if otherConn != conn && len(otherClient.team) > 0 {
            fmt.Fprintln(otherConn, fmt.Sprintf("%s wins!", client.conn.RemoteAddr().String()))
        }
    }
}

func main() {
    rand.Seed(time.Now().UnixNano())
    FetchAllPokemonData()

    pokedexFile, err := os.Open("pokedex.json")
    if err != nil {
        log.Fatalf("Error opening pokedex.json: %v", err)
    }
    defer pokedexFile.Close()

    var pokedex []Pokemon
    if err := json.NewDecoder(pokedexFile).Decode(&pokedex); err != nil {
        log.Fatalf("Error decoding pokedex.json: %v", err)
    }

    listener, err := net.Listen("tcp", ":8080")
    if err != nil {
        log.Fatalf("Error listening: %v", err)
    }
    defer listener.Close()

    fmt.Println("Server listening on port 8080...")

    var player1Conn, player2Conn net.Conn
    clients := make(map[net.Conn]*Client)

    for {
        conn, err := listener.Accept()
        if err != nil {
            log.Printf("Error accepting connection: %v", err)
            continue
        }

        if player1Conn == nil {
            fmt.Println("Player 1 connected!")
            player1Conn = conn
        } else if player2Conn == nil {
            fmt.Println("Player 2 connected!")
            player2Conn = conn
            break
        }
    }

    done := make(chan struct{}, 2)

    go handleConnection(player1Conn, pokedex, clients, done)
    go handleConnection(player2Conn, pokedex, clients, done)

    <-done
    <-done

    fmt.Println("Both players are ready! Let the battle begin!")

    // Determine which player has the Pokémon with higher speed
    player1Speed := clients[player1Conn].team[0].Speed
    player2Speed := clients[player2Conn].team[0].Speed

    if player1Speed > player2Speed {
        clients[player1Conn].isActive = true
        fmt.Fprintln(player1Conn, "Your Pokémon is faster, you go first!")
        fmt.Fprintln(player2Conn, "Opponent's Pokémon is faster. They go first!")
    } else if player2Speed > player1Speed {
        clients[player2Conn].isActive = true
        fmt.Fprintln(player1Conn, "Your Pokémon is faster, you go first!")
        fmt.Fprintln(player2Conn, "Opponent's Pokémon is faster. They go first!")
    } else {
        // If speeds are equal, randomly choose which player goes first
        if rand.Intn(2) == 0 {
            clients[player1Conn].isActive = true
            fmt.Fprintln(player1Conn, "Your Pokémon is faster, you go first!")
            fmt.Fprintln(player2Conn, "Opponent's Pokémon is faster. They go first!")
        } else {
            clients[player2Conn].isActive = true
            fmt.Fprintln(player1Conn, "Your Pokémon is faster, you go first!")
            fmt.Fprintln(player2Conn, "Opponent's Pokémon is faster. They go first!")
        }
    }

    select {} // keep server running
}
