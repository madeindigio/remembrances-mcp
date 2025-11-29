Quiero que uses las tools de remembrances para conocer información almacenada que pueda estar relacionado con la tarea que te pueda mandar, y cuando
termines una tarea larga, usa estas tools para guardar cualquier información que consideres tener para el futuro.

Ahora, quuiero que planifiques la siguiente tarea larga, genera un plan en el fichero .serena/memories/plan.md para desglosarlo en diferentes subtareas:

Quiero soportar la lectura de proyectos de código de programación, para esto quiero usar la librería tree-sitter para soportar los lenguajes de
programación siguientes: php, typescript, javascript, golang, rust, java, kotlin, swift y objective-c . Quiero poder indexar el código de un proyecto en
 la base de datos de surrealdb, conectada, para esto quiero tener tablas de datos separadas, que identifiquen el proyecto por un nombre de proyecto,
deberá de tener un identificador de proyecto estas tablas. La idea es poder indexar a través de su arbol AST con tree-sitter y generar embeddings de
este código, almacenado en vectores en esta tabla, para posteriormente implementar funciones de búsqueda híbrida por embeddings, o nombre de clases,
métodos, funciones etc...

Quiero disponer de un argumento por línea de comandos, que adicionalmente a la configuración, dando un argumento con el identificador de proyecto, le
pueda pasar un path, para indexar este código en base de datos y generar sus embeddings.

Luego quiero tener una tool mcp para poder hacer lo mismo pero que no espere terminar la indexación para dar una respuesta lanzada la indexación en
segundo plano, si vuelvo a llamar a la tool debo de detectar si he acabado la indexación o está en curso, y devolverlo como respuesta, no lanzar la
indexación nueva en paralelo si ya se está haciendo.

Cuando tengamos esta parte hecha, quiero tener disponible tools para búsqueda sobre el código, quiero implementar unas tools similares a las de Serena
MCP (busca en internet, te paso una referencia https://github.com/oraios/serena/tree/main/src/serena/tools ), no implementes las tools de memoria (ya
las contiene Remembrances), sólamente la de búsqueda de símbolos, escritura y reemplazo de código. Deberá tener tools con métodos equivalentes pero la
implementación a realizar es la descrita en esta tarea, no cómo lo hace Serena.

Cuando termines de documentarte y plantear el plan, espera que lo revise, para darte los siguientes pasos
