<img src="https://r2cdn.perplexity.ai/pplx-full-logo-primary-dark%402x.png" style="height:64px;margin-right:32px"/>

# quiero parsear en Golang una estructura Jason donde algunos campos tienen una estructura definida y otros campos como metadata, no la tienen. además quiero guardar esta metadata en un campo de surrealdb, analiza que tipo de campo debería de utilizar para almacenar documentos no estructurados como Jason y al mismo tiempo poder hacer búsquedas dentro de los mismos

Puedes parsear una estructura JSON en Go donde algunos campos son definidos y otros como metadata son libres, utilizando estructuras que incluyan campos del tipo map[string]interface{} para esa metadata. Para almacenar este JSON no estructurado en SurrealDB y poder buscar dentro de esos datos, lo recomendable es:

### Tipo de campo y definición para metadata

Usa un campo definido como object FLEXIBLE en SurrealDB. Esto permite que tu campo metadata acepte cualquier estructura JSON, sin restricciones, y SurrealDB lo mantiene como un objeto flexible. Ejemplo de definición en SurrealQL:

```
DEFINE FIELD metadata ON TABLE user TYPE object FLEXIBLE;
```

Esto hará que SurrealDB almacene correctamente cualquier JSON en ese campo, aunque cada registro tenga diferente estructura en metadata[^1_1][^1_2][^1_3].

### Búsqueda dentro del campo metadata

SurrealDB permite hacer búsquedas dentro de campos tipo object utilizando la sintaxis de acceso a subcampos (dot notation) en tus consultas. Por ejemplo, puedes buscar todos los usuarios cuyo metadata tenga el país “ES” así:

```
SELECT * FROM user WHERE metadata.country_code = "ES";
```

Esto funciona porque los objetos FLEXIBLE permiten almacenar datos no estructurados y consultarlos dinámicamente por subcampos[^1_2][^1_4][^1_3].

### Consideraciones para integración en Go

En Go, el struct podría verse así:

```go
type User struct {
    Name     string                 `json:"name"`
    Metadata map[string]interface{} `json:"metadata"`
}
```

Esto te permite parsear el JSON con campos fijos (como Name) y un campo metadata flexible, compatible con lo que espera SurrealDB para el almacenamiento y las búsquedas posteriores.

### Resumen de recomendaciones

- En SurrealDB, usa TYPE object FLEXIBLE y así puedes guardar cualquier JSON como metadata.
- Las búsquedas internas usan notación de puntos en SurrealQL.
- En Go, utiliza map[string]interface{} para los campos flexibles.

Con esto tienes máxima flexibilidad y capacidad de búsqueda sobre estructuras JSON no definidas en SurrealDB[^1_1][^1_2][^1_3].
<span style="display:none">[^1_10][^1_11][^1_12][^1_13][^1_14][^1_15][^1_16][^1_17][^1_18][^1_19][^1_20][^1_5][^1_6][^1_7][^1_8][^1_9]</span>

<div align="center">⁂</div>

[^1_1]: https://surrealdb.com/blog/ten-tips-and-tricks-for-your-database-schema

[^1_2]: https://surrealdb.com/docs/surrealql/statements/define/field

[^1_3]: https://surrealdb.com/docs/tutorials/define-a-schema

[^1_4]: https://surrealdb.com/docs/surrealql/statements/select

[^1_5]: https://surrealdb.com/docs/surrealdb/models/document

[^1_6]: https://learn.microsoft.com/es-es/sql/relational-databases/json/store-json-documents-in-sql-tables?view=sql-server-ver17

[^1_7]: https://surrealdb.com/docs/surrealml/storage

[^1_8]: https://surrealdb.com/docs/integrations/data-management/surreal-sync/json-lines

[^1_9]: https://learn.microsoft.com/es-es/sql/relational-databases/json/json-data-sql-server?view=sql-server-ver17

[^1_10]: https://surrealdb.com/docs/surrealql/functions/script/built-in-functions

[^1_11]: https://surrealdb.com/learn/fundamentals/schemaless

[^1_12]: https://www.reddit.com/r/webdev/comments/1gnc5dg/how_do_you_decide_between_using_sql_and_nosql/

[^1_13]: https://surrealdb.com/learn/tour/page-17

[^1_14]: https://upcommons.upc.edu/bitstreams/cb8f0df2-538d-404d-9842-848dd6e0c8e0/download

[^1_15]: https://surrealdb.com/docs/surrealdb/models

[^1_16]: https://stackoverflow.com/questions/78625030/in-surrealdb-sql-how-do-you-find-and-update-value-in-array-of-objects

[^1_17]: https://docs.rs/crate/surrealdb/latest

[^1_18]: https://www.youtube.com/shorts/dfBSFrfovZI

[^1_19]: https://surrealdb.com/docs/surrealdb/querying

[^1_20]: https://surrealdb.com/docs/integrations/data-management/unstructured


---

# como podría luego parsear este tipo de objetos json desde Golang dame ejemplos de cómo funcionan

Puedes parsear y manejar objetos JSON dinámicos o no estructurados en Go usando map[string]interface{}. Esto te permite acceder y trabajar con datos flexibles como los que guardarías en el campo metadata de SurrealDB. Aquí tienes ejemplos prácticos de cómo hacerlo:

### Parsear JSON a map[string]interface{}

Supongamos que tienes un JSON como este:

```json
{
  "nombre": "Ana",
  "metadata": {
    "edad": 33,
    "pais": "ES",
    "preferencias": {
      "idioma": "español",
      "likes": ["música", "lectura"]
    }
  }
}
```

Código para parsear en Go:

```go
package main

import (
    "encoding/json"
    "fmt"
)

func main() {
    var data map[string]interface{}
    input := `{
        "nombre": "Ana",
        "metadata": {
            "edad": 33,
            "pais": "ES",
            "preferencias": {
                "idioma": "español",
                "likes": ["música", "lectura"]
            }
        }
    }`
    err := json.Unmarshal([]byte(input), &data)
    if err != nil {
        panic(err)
    }
    fmt.Println(data)
}
```


### Acceder a campos flexibles en metadata

Puedes acceder a subcampos usando conversiones de tipo:

```go
metadata := data["metadata"].(map[string]interface{})
edad := metadata["edad"].(float64)  // Go convierte números JSON en float64 por defecto
pais := metadata["pais"].(string)

prefs := metadata["preferencias"].(map[string]interface{})
idioma := prefs["idioma"].(string)
likes := prefs["likes"].([]interface{})
fmt.Println(edad, pais, idioma, likes)
```


### Consejos y casos prácticos

- Siempre comprueba la presencia del campo y su tipo con type assertions para evitar panics.
- Si necesitas trabajar con enteros grandes, puedes usar json.NewDecoder().UseNumber() para evitar conversiones automáticas a float64[^2_1].

Ejemplo para mantener números como json.Number:

```go
import "encoding/json"
import "bytes"

jsonData := `{ "valor": 9223372036854775807 }`
decoder := json.NewDecoder(bytes.NewReader([]byte(jsonData)))
decoder.UseNumber()

var result map[string]interface{}
decoder.Decode(&result)

valor := result["valor"].(json.Number)
fmt.Println(valor.String())
```

Este patrón se adapta perfectamente a los campos flexibles tipo metadata en SurrealDB, permitiéndote tanto guardar como consultar según la estructura concreta que tenga cada registro.
<span style="display:none">[^2_10][^2_11][^2_12][^2_13][^2_14][^2_15][^2_16][^2_17][^2_18][^2_19][^2_2][^2_20][^2_3][^2_4][^2_5][^2_6][^2_7][^2_8][^2_9]</span>

<div align="center">⁂</div>

[^2_1]: https://stackoverflow.com/questions/47400358/go-converting-json-string-to-mapstringinterface

[^2_2]: https://gobyexample.com/json

[^2_3]: https://www.reddit.com/r/golang/comments/1bu7zps/parsing_json_in_order_as_a_map_stringinterface/

[^2_4]: https://tutorialedge.net/golang/parsing-json-with-golang/

[^2_5]: https://www.golinuxcloud.com/golang-parse-json/

[^2_6]: https://betterstack.com/community/guides/scaling-go/json-in-go/

[^2_7]: https://blog.codeutility.io/es/programming/how-to-use-arrays-in-json-with-examples-in-code-5eeef2665f

[^2_8]: https://eli.thegreenplace.net/2020/representing-json-structures-in-go/

[^2_9]: https://stackoverflow.com/questions/19691468/json-decode-with-flexible-type

[^2_10]: https://cloud.google.com/spanner/docs/working-with-json?hl=es-419

[^2_11]: https://gist.github.com/cuixin/f10cea0f8639454acdfbc0c9cdced764

[^2_12]: https://dev.to/arshamalh/how-to-unmarshal-json-in-a-custom-way-in-golang-42m5

[^2_13]: https://stackoverflow.com/questions/35660467/how-to-access-fields-of-a-json-in-go

[^2_14]: https://www.geeksforgeeks.org/go-language/how-to-parse-json-in-golang/

[^2_15]: https://pkg.go.dev/encoding/json/v2

[^2_16]: https://codesignal.com/learn/courses/handling-json-in-go-1/lessons/working-with-dynamic-or-unknown-json-structures-in-go

[^2_17]: https://yourbasic.org/golang/json-example/

[^2_18]: https://www.youtube.com/watch?v=76aORU14cyY

[^2_19]: https://golang.cafe/blog/golang-json-marshal-example.html

[^2_20]: https://www.youtube.com/watch?v=a96veXdifys


---

# como podría hacerse una librería que pueda leer campos fijos y campos dinámicos para devolver un tipo de datos Golang

Te muestro cómo crear una librería que maneje campos fijos y dinámicos, devolviendo un tipo de datos Go bien definido. La clave está en implementar el método `UnmarshalJSON` personalizado y usar `json.RawMessage` para procesar campos dinámicos[^3_1][^3_2].

## Enfoque: Struct con UnmarshalJSON personalizado

### Definición de la estructura base

```go
package flexjson

import (
    "encoding/json"
    "fmt"
)

// FlexData representa datos con campos fijos y dinámicos
type FlexData struct {
    // Campos fijos
    ID        string    `json:"id"`
    Name      string    `json:"name"`
    CreatedAt string    `json:"created_at"`
    
    // Campos dinámicos (metadata)
    Metadata  map[string]interface{} `json:"-"`
}
```


### Implementación de UnmarshalJSON personalizado

```go
// UnmarshalJSON implementa deserialización personalizada
func (f *FlexData) UnmarshalJSON(data []byte) error {
    // Paso 1: Parsear campos fijos usando un struct auxiliar
    type Alias FlexData
    aux := &struct {
        *Alias
    }{
        Alias: (*Alias)(f),
    }
    
    if err := json.Unmarshal(data, &aux); err != nil {
        return err
    }
    
    // Paso 2: Parsear todo el JSON a un mapa
    var raw map[string]interface{}
    if err := json.Unmarshal(data, &raw); err != nil {
        return err
    }
    
    // Paso 3: Extraer campos dinámicos (los que no son fijos)
    f.Metadata = make(map[string]interface{})
    knownFields := map[string]bool{
        "id":         true,
        "name":       true,
        "created_at": true,
    }
    
    for key, value := range raw {
        if !knownFields[key] {
            f.Metadata[key] = value
        }
    }
    
    return nil
}
```


### Implementación de MarshalJSON para serialización

```go
// MarshalJSON implementa serialización personalizada
func (f FlexData) MarshalJSON() ([]byte, error) {
    // Crear mapa con campos fijos
    result := map[string]interface{}{
        "id":         f.ID,
        "name":       f.Name,
        "created_at": f.CreatedAt,
    }
    
    // Añadir campos dinámicos
    for key, value := range f.Metadata {
        result[key] = value
    }
    
    return json.Marshal(result)
}
```


## Enfoque alternativo: Usando json.RawMessage

Este enfoque es más eficiente cuando quieres procesar campos dinámicos de forma diferida:

```go
type FlexDataRaw struct {
    ID        string          `json:"id"`
    Name      string          `json:"name"`
    CreatedAt string          `json:"created_at"`
    Extra     json.RawMessage `json:"-"` // Contiene el resto sin procesar
}

func (f *FlexDataRaw) UnmarshalJSON(data []byte) error {
    type Alias FlexDataRaw
    aux := &struct{ *Alias }{Alias: (*Alias)(f)}
    
    if err := json.Unmarshal(data, &aux); err != nil {
        return err
    }
    
    // Guardar todo el JSON sin procesar
    var all map[string]json.RawMessage
    if err := json.Unmarshal(data, &all); err != nil {
        return err
    }
    
    // Crear nuevo mapa excluyendo campos conocidos
    extra := make(map[string]json.RawMessage)
    for k, v := range all {
        if k != "id" && k != "name" && k != "created_at" {
            extra[k] = v
        }
    }
    
    // Serializar solo los campos extra
    f.Extra, _ = json.Marshal(extra)
    return nil
}

// Método helper para acceder a metadata
func (f *FlexDataRaw) GetMetadata() (map[string]interface{}, error) {
    var result map[string]interface{}
    err := json.Unmarshal(f.Extra, &result)
    return result, err
}
```


## Uso de la librería

```go
package main

import (
    "encoding/json"
    "fmt"
    "log"
)

func main() {
    jsonData := `{
        "id": "123",
        "name": "Usuario Test",
        "created_at": "2025-10-25",
        "edad": 30,
        "pais": "ES",
        "preferencias": {
            "tema": "oscuro",
            "idioma": "español"
        }
    }`
    
    var data FlexData
    if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("ID: %s\n", data.ID)
    fmt.Printf("Name: %s\n", data.Name)
    fmt.Printf("Metadata: %+v\n", data.Metadata)
    
    // Acceder a campos dinámicos
    if edad, ok := data.Metadata["edad"].(float64); ok {
        fmt.Printf("Edad: %.0f\n", edad)
    }
    
    // Serializar de vuelta a JSON
    output, _ := json.Marshal(data)
    fmt.Printf("JSON: %s\n", string(output))
}
```


## Ventajas de este diseño

1. **Tipo seguro**: Los campos fijos tienen tipos definidos
2. **Flexible**: Metadata admite cualquier estructura
3. **Compatible con SurrealDB**: Puedes guardar el campo `Metadata` como `object FLEXIBLE`
4. **Bidireccional**: Soporta serialización y deserialización
5. **Sin dependencias externas**: Solo usa `encoding/json` estándar

Este patrón es ideal para tu caso de uso con SurrealDB, donde necesitas parsear JSON con estructura mixta y almacenar metadata flexible[^3_1][^3_3][^3_2].
<span style="display:none">[^3_10][^3_11][^3_12][^3_13][^3_14][^3_15][^3_16][^3_17][^3_18][^3_19][^3_20][^3_4][^3_5][^3_6][^3_7][^3_8][^3_9]</span>

<div align="center">⁂</div>

[^3_1]: https://pkg.go.dev/encoding/json

[^3_2]: https://stackoverflow.com/questions/33436730/unmarshal-json-with-some-known-and-some-unknown-field-names

[^3_3]: https://victoronsoftware.com/posts/go-json-unmarshal/

[^3_4]: https://gist.github.com/Integralist/d47c2e8c6064ec065108ad59df6e1fb9

[^3_5]: https://stackoverflow.com/questions/39223754/unmarshalling-json-to-structure-using-json-rawmessage

[^3_6]: https://www.reddit.com/r/golang/comments/17bqurd/how_would_you_unmarshall_json_where_a_member_can/

[^3_7]: https://boldlygo.tech/posts/2019-12-09-slightly-custom-marshaler/

[^3_8]: https://www.golinuxcloud.com/golang-parse-json/

[^3_9]: https://gist.github.com/ik5/a4521a4166302efecc3d3f8ea8080912

[^3_10]: https://stackoverflow.com/questions/74762073/handling-nested-unstructured-json-in-go-lang

[^3_11]: https://www.sohamkamani.com/golang/json/

[^3_12]: https://www.reddit.com/r/golang/comments/nppxma/how_to_json_unmarshal_field_of_dynamic_type/

[^3_13]: https://www.reddit.com/r/golang/comments/z6iu1n/is_there_a_way_to_parse_unstructured_data/

[^3_14]: https://labex.io/tutorials/go-how-to-encode-custom-types-to-json-431212

[^3_15]: https://pedroperez.dev/posts/2023/03/interfaces-en-go-ii-implementación-de-unmarshaler/

[^3_16]: https://github.com/buger/jsonparser

[^3_17]: https://dev.to/arshamalh/how-to-unmarshal-json-in-a-custom-way-in-golang-42m5

[^3_18]: https://blog.min.io/simdjson-go-parsing-gigabyes-of-json-per-second-in-go/

[^3_19]: https://golang.cafe/blog/golang-json-marshal-example.html

[^3_20]: https://tutorialedge.net/golang/parsing-json-with-golang/

