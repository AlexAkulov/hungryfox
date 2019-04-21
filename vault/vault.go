package vault

import (
	"fmt"
	"strings"

	"github.com/AlexAkulov/hungryfox/config"
	"github.com/hashicorp/vault/api"
)

type vaultPath struct {
	Mount string
	v2    bool
	Path  string
}

func (vp vaultPath) List() string {
	if vp.v2 {
		return strings.Join([]string{vp.Mount, "metadata", vp.Path}, "/")
	}
	return strings.Join([]string{vp.Mount, vp.Path}, "/")
}

func (vp vaultPath) Read() string {
	if vp.v2 {
		return strings.Join([]string{vp.Mount, "data", vp.Path}, "/")
	}
	return strings.Join([]string{vp.Mount, vp.Path}, "/")
}

func (vp vaultPath) String() string {
	return vp.Mount + "/" + vp.Path
}

func toVaultPath(path string, v2 bool) vaultPath {
	vp := vaultPath{}
	path = strings.TrimPrefix(path, "/")
	part := strings.SplitN(path, "/", 2)
	vp.Mount = part[0]
	if len(part) > 1 {
		vp.Path = part[1]
	}
	vp.v2 = v2
	return vp
}

type Vault struct {
	client *api.Client
	token  string
	Config *config.Vault
}

func (v *Vault) checkSecretEngine(path string) (vaultPath, error) {
	vp := toVaultPath(path, false)
	secret, err := v.client.Logical().List(vp.List())
	if err == nil || secret != nil {
		return vp, nil
	}
	vp.v2 = true
	secret, err = v.client.Logical().List(vp.List())
	if err != nil {
		return vp, err
	}
	return vp, nil
}

func (v *Vault) Start() error {
	var err error
	if v.client, err = api.NewClient(
		&api.Config{Address: v.Config.VaultURL}); err != nil {
		return err
	}
	v.client.SetToken(v.Config.Token)

	return nil
}

func (v *Vault) Stop() error {
	return nil
}

func (v *Vault) list(vp vaultPath) ([]vaultPath, error) {
	secret, err := v.client.Logical().List(vp.List())
	if err != nil {
		return nil, err
	}
	if secret == nil {
		return nil, nil
	}
	responce, ok := secret.Data["keys"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("no result")
	}
	out := make([]vaultPath, len(responce))
	for i := range responce {
		out[i] = vaultPath{
			Mount: vp.Mount,
			Path:  vp.Path + responce[i].(string),
			v2:    vp.v2,
		}
	}
	return out, nil
}

func (v *Vault) listAll(vp vaultPath) []vaultPath {
	result, err := v.list(vp)
	if err != nil {
		return nil
	}
	newr := result
	if result != nil {
		for _, k := range result {
			if !strings.HasSuffix(k.Path, "/") {
				continue
			}
			if r := v.listAll(k); r != nil {
				newr = append(newr, r...)
			}
		}
	}
	return newr
}

func (v *Vault) readAll(path string) (map[string]string, error) {
	rootPath, err := v.checkSecretEngine(path)
	if err != nil {
		return nil, err
	}
	vps := v.listAll(rootPath)
	result := map[string]string{}
	for _, vp := range vps {
		if strings.HasSuffix(vp.Path, "/") {
			continue
		}
		secrets, err := v.read(vp)
		if err != nil {
			return nil, err
		}
		for name, secret := range secrets {
			if str, ok := secret.(string); ok {
				result[vp.String()+":"+name] = str
			}
		}
	}
	return result, nil
}

func (v *Vault) ReadAll() (map[string]string, error) {
	result := make(map[string]string)
	for _, path := range v.Config.Paths {
		r, err := v.readAll(path)
		if err != nil {
			return nil, err
		}
		for k, v := range r {
			result[k] = v
		}
	}
	return result, nil
}

func (v *Vault) read(vp vaultPath) (map[string]interface{}, error) {
	out := make(map[string]interface{})
	secret, err := v.client.Logical().Read(vp.Read())
	if err != nil {
		return nil, fmt.Errorf("can't read secret with: %v", err)
	}
	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("No data to read at path, %s", vp.Read())
	}
	for k, v := range secret.Data {
		switch t := v.(type) {
		case string:
			// out[k] = base64.StdEncoding.EncodeToString([]byte(t))
			out[k] = v.(string)
		case map[string]interface{}:
			if k == "data" {
				for x, y := range t {
					if z, ok := y.(string); ok {
						// out[x] = base64.StdEncoding.EncodeToString([]byte(z))
						out[x] = z
					}
				}
			}
		default:
			return nil, fmt.Errorf("error reading value at %s, key=%s, type=%T", vp.Read(), k, v)
		}
	}
	return out, nil
}

// func main() {
// 	v := Vault{
// 		Config: &config.Vault{
// 			VaultURL: "https://vault",
// 			Token:    "...",
// 		},
// 	}
// 	if err := v.Start(); err != nil {
// 		fmt.Println("can't start vault")
// 		fmt.Println(err)
// 		os.Exit(1)
// 	}
// 	result := v.ListAll(toVaultPath("secret/", true))
// 	if result == nil {
// 		fmt.Println("can't get secrets path")
// 		os.Exit(1)
// 	}
// 	secrets, err := v.ReadAll(result)
// 	if err != nil {
// 		fmt.Println("can't get secrets")
// 		fmt.Println(err)
// 		os.Exit(1)
// 	}
// 	for k, v := range secrets {
// 		fmt.Printf("path=%v\t\tvalue=%v\n", k, v)
// 	}
// }
