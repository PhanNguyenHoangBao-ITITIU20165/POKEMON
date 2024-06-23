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
    "sync"
    "time"
    "io/ioutil"
)
// Player represents a player in the game world with their attributes and Pokémon collection.
type Player struct {
	ID       int
	X, Y     int
	Pokemons map[int]Pokemon // Map of Pokémon owned by the player
	mutex    sync.Mutex // Mutex for synchronizing access to player data
}

var (
    pokeworld *Pokeworld
	world       [][]*Player
	worldMutex  sync.Mutex
	playerID    int
	playerMutex sync.Mutex
	players     []*Player
)

const (
	worldSizeX          = 1000
	worldSizeY          = 1000
	autoModeDurationSec = 120
	pokemonSpawnRate    = 1 * time.Minute
	pokemonDespawnTime  = 5 * time.Minute
	maxPokemonPerPlayer = 200
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
    Level          int      `json:"level"`
    AccumExp       int      `json:"accum_exp"`
    EV             float64  `json:"ev"`  
    Owner          *Client  
}


type Pokeworld struct {
    grid               [][]*Pokemon     // Grid representing the game world
    players            map[net.Conn]*Client // Map of active players
    pokedex            []Pokemon        // List of all available Pokémon
    Width, Height      int              // Dimensions of the game world
    PokemonSpawnRate   time.Duration    // Rate at which Pokémon spawn
    PokemonDespawnTime time.Duration    // Time after which Pokémon despawn
    PokemonPerSpawn    int              // Number of Pokémon to spawn at once
    rng                *rand.Rand       // Random number generator for spawning Pokémon
    NextSpawn          time.Time        // Next time Pokémon will spawn
    despawnTimers      map[*Pokemon]*time.Timer // Timers for despawning Pokémon
    TotalPokemon       int              // Total number of Pokémon currently in the world
    sync.Mutex                          // Mutex for synchronizing access to Pokeworld data
}

type Client struct {
    conn          net.Conn       // Connection object for the client
    team          []*Pokemon     // Team of Pokémon selected by the client
    isActive      bool           // Indicates if the client is actively participating in a battle
    activePokemon *Pokemon       // Active Pokémon selected for battle
    reader        *bufio.Reader  // Reader for reading input from the client
    X, Y          int            // Coordinates of the client in the game world
    AutoMode      bool           // Indicates if the client is in auto mode
    AutoUntil     time.Time      // Time until which auto mode is active
    sync.Mutex                   // Mutex for synchronizing access to client data
}

// TypeEffectiveness represents the effectiveness multiplier of one type against another.
type TypeEffectiveness struct {
    AttackingType string  `json:"attacking_type"`
    DefendingType string  `json:"defending_type"`
    Multiplier    float64 `json:"multiplier"`
}

// typeChart represents the effectiveness chart of Pokémon types against each other.
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
//Pokedex
// fetchPokemonData fetches Pokemon data from the PokeAPI for the given ID.
func fetchPokemonData(id int) (*Pokemon, error) {
    // Construct the PokeAPI URL using the provided Pokemon ID.
    resp, err := http.Get(fmt.Sprintf("https://pokeapi.co/api/v2/pokemon/%d/", id))
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    // Create a generic map to store the raw JSON data.
    var result map[string]interface{}
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil { // Decode the JSON response into the 'result' map.
        return nil, err
    }

    // Extract and process types from the JSON response.
    types := []string{}
    for _, t := range result["types"].([]interface{}) {
        typeName := t.(map[string]interface{})["type"].(map[string]interface{})["name"].(string)
        types = append(types, typeName)
    }

    // Extract and process base stats from the JSON response.
    stats := make(map[string]int)
    for _, s := range result["stats"].([]interface{}) {
        statName := s.(map[string]interface{})["stat"].(map[string]interface{})["name"].(string)
        statValue := int(s.(map[string]interface{})["base_stat"].(float64))
        stats[statName] = statValue
    }

    // Create a Pokemon struct and populate its fields with extracted data.
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
        Level:          1,
        AccumExp:       0,
        EV:             0.5, 
    }

    return pokemon, nil
}

func FetchAllPokemonData() {
    var pokemons []Pokemon // Create a slice to store all Pokémon data.

    for i := 1; i <= 200; i++ { // Fetch first 200 Pokémon
        pokemon, err := fetchPokemonData(i)
        if err != nil {
            log.Printf("Error fetching data for Pokémon ID %d: %v", i, err)
            continue
        }
        pokemons = append(pokemons, *pokemon)
    }

    // Create the JSON file.
    jsonFile, err := os.Create("pokedex.json")
    if err != nil {
        log.Fatal(err)
    }
    defer jsonFile.Close()

    // Create a JSON encoder with indentation for readability.
    encoder := json.NewEncoder(jsonFile)
    encoder.SetIndent("", "  ")
    err = encoder.Encode(pokemons)
    // Encode the Pokémon data into the JSON file.
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("Pokédex successfully created!")
}

//Pokecat
func (pw *Pokeworld) spawnPokemonLoop() {
    for {
        time.Sleep(pokemonSpawnRate) // Spawn every minute
        pw.spawnPokemon(50) // Spawn 50 Pokémon
    }
}

// spawnPokemon generates new Pokémon in the Pokeworld.
func (pw *Pokeworld) spawnPokemon(numPokemon int) {
    // Ensure exclusive access to the Pokeworld during spawning.
    pw.Lock()
    defer pw.Unlock()

    // Loop to spawn the specified number of Pokemon (up to the world limit of 50).
    for i := 0; i < numPokemon && pw.TotalPokemon < 50; i++ {
        // Generate random coordinates within the world.
        x, y := rand.Intn(worldSizeX), rand.Intn(worldSizeY)
        var pokemon *Pokemon

        // Check if the grid cell is empty.
        if pw.grid[x][y] == nil {
            // Randomly select a Pokemon from the pokedex. 
            pokemonIndex := rand.Intn(len(pw.pokedex)) // Assign the selected Pokémon
            pokemon = &pw.pokedex[pokemonIndex] 
            // Set random level and EV for the Pokemon.
            pokemon.Level = rand.Intn(100) + 1
            pokemon.EV = rand.Float64()*0.5 + 0.5
            pw.grid[x][y] = pokemon

            //start a timer for despawning this Pokémon
            despawnTimer := time.AfterFunc(pw.PokemonDespawnTime, func() {
                pw.Lock()
                defer pw.Unlock()

                // Ensure the Pokémon still exists before despawning
                if pw.grid[x][y] == pokemon {
                    pw.grid[x][y] = nil
                    delete(pw.despawnTimers, pokemon)
                    pw.TotalPokemon--
                }
            })
            pw.despawnTimers[pokemon] = despawnTimer // Store the timer
        }
    }
}

// addPlayer creates a new player at the specified coordinates (x, y), 
// adds them to the global player list, and places them in the game world.
// It returns a pointer to the newly created player.
func addPlayer(x, y int) *Player {
    // Ensure exclusive access to the player list while modifying it.
	playerMutex.Lock()
	defer playerMutex.Unlock() // Unlock when the function exits.

	playerID++
    // Create a new Player struct with the assigned ID, position, and empty Pokemons map.
	player := &Player{
		ID:       playerID,
		X:        x,
		Y:        y,
		Pokemons: make(map[int]Pokemon),
	}

	players = append(players, player) // Add the player to the global slice of players.
	worldMutex.Lock() // Ensure exclusive access to the game world while placing the player.
	world[x][y] = player //Update the game world grid to place the player at the specified location.
	worldMutex.Unlock()

	return player
}

func handlePlayerMovement(player *Player, direction string) {
	player.mutex.Lock()
	defer player.mutex.Unlock()

	worldSize := len(world) // Get world dimensions

	// Store original position (for potential reset due to out-of-bounds)
	originalX, originalY := player.X, player.Y

	switch direction {
	case "up":
		player.Y = (player.Y - 1 + worldSize) % worldSize // Wrap around if at top edge
	case "down":
		player.Y = (player.Y + 1) % worldSize // Wrap around if at bottom edge
	case "left":
		player.X = (player.X - 1 + worldSize) % worldSize // Wrap around if at left edge
	case "right":
		player.X = (player.X + 1) % worldSize // Wrap around if at right edge
	default:
		return // Invalid direction, do nothing
	}

	// Check if new position is already occupied by another player
	worldMutex.Lock()
	if world[player.X][player.Y] != nil && len(world[player.X][player.Y].Pokemons) == 0 {
		// Reset to original position if occupied
		player.X, player.Y = originalX, originalY
		worldMutex.Unlock()
		return
	}

	// Update world
	world[originalX][originalY] = nil
	world[player.X][player.Y] = player
	worldMutex.Unlock()

	// Check for Pokemon encounter
	if pokemon := world[player.X][player.Y]; pokemon != nil && len(pokemon.Pokemons) > 0 {
		handleEncounter(player, pokemon)
	}
}

func movePlayer(player *Player, direction string) {
	handlePlayerMovement(player, direction)
}

func handleEncounter(player *Player, pokemon *Player) {
	player.mutex.Lock()
	defer player.mutex.Unlock()

	worldMutex.Lock()
	defer worldMutex.Unlock()

	if len(player.Pokemons) < 200 {
		// Auto-capture
		for pokemonId, pokemonData := range pokemon.Pokemons {
			player.Pokemons[pokemonId] = pokemonData
		}
		world[player.X][player.Y] = nil // Remove from world

		// Save player data (update their file)
		saveData(player, fmt.Sprintf("player_data/player%d_data.json", player.ID))
	}
}

func generateRandomEVs() []float64 {
	EVs := make([]float64, 6)
	for i := range EVs {
		EVs[i] = rand.Float64()*(1-0.5) + 0.5
	}
	return EVs
}

// Save player data
func saveData(player *Player, filename string) {
	data, _ := json.MarshalIndent(player, "", "  ")
	_ = ioutil.WriteFile(filename, data, 0644)
}

// Additional helper functions
func getAllPlayers() []*Player {
	worldMutex.Lock()
	defer worldMutex.Unlock()
	return players
}

func randomDirection() string {
	directions := []string{"up", "down", "left", "right"}
	return directions[rand.Intn(len(directions))]
}

//Pokebat
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

// handleConnection manages a single client connection in the Pokémon battle.
func handleConnection(conn net.Conn, pokedex []Pokemon, clients map[net.Conn]*Client, done chan struct{}) {
    defer conn.Close()

    // Create a new client struct for the connection and add to the clients map.
    client := &Client{conn: conn, isActive: false, reader: bufio.NewReader(conn)}
    clients[conn] = client

    fmt.Fprintln(conn, "Welcome to the Pokémon battle!")
    fmt.Fprintln(conn, "Choose 3 Pokémon for your team:")
    for i, p := range pokedex {
        fmt.Fprintf(conn, "%d. %s (%v)\n", i+1, p.Name, p.Type)
    }
    // Loop to get the player's team choices.
    for i := 0; i < 3; i++ {
        fmt.Fprint(conn, "Enter number for Pokémon: ")
        input, err := client.reader.ReadString('\n') // Read the player's choice.
        if err != nil {
            fmt.Println("Error reading from client:", err)
            return
        }
        input = strings.TrimSpace(input)

        // Convert input to integer and validate the choice.
        choice, err := strconv.Atoi(input)
        if err != nil || choice < 1 || choice > len(pokedex) {
            fmt.Fprintln(conn, "Invalid choice. Try again.")
            i--
            continue
        }

        // Add the chosen Pokémon to the client's team.
        client.team = append(client.team, &pokedex[choice-1])
    }

    // Notify the main server that a client is ready (using the 'done' channel).
    done <- struct{}{}

    for {
        // Check if it's this client's turn and if they have Pokémon remaining.
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
            time.Sleep(4 * time.Second) // Adjust wait time as needed
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

    // Open the saved Pokedex data from the JSON file.
    pokedexFile, err := os.Open("pokedex.json")
    if err != nil {
        log.Fatalf("Error opening pokedex.json: %v", err)
    }
    defer pokedexFile.Close()

    // Decode the JSON data into a slice of Pokemon structs.
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

    // Wait for two players to connect.
    for {
        conn, err := listener.Accept()
        if err != nil {
            log.Printf("Error accepting connection: %v", err)
            continue
        }

        //go pokeworld.handlePlayer(conn)

        //Assign connection to player1 or player2
        if player1Conn == nil {
            fmt.Println("Player 1 connected!")
            player1Conn = conn
        } else if player2Conn == nil {
            fmt.Println("Player 2 connected!")
            player2Conn = conn
            break
        }
    }

    //Channel to signal when both players are ready to start.
    done := make(chan struct{}, 2)

    // Start handling connections for both players concurrently.
    go handleConnection(player1Conn, pokedex, clients, done)
    go handleConnection(player2Conn, pokedex, clients, done)

    <-done // Wait for both players to be ready before starting the battle.
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
