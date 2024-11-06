# c# 之旅

C#语言是[.NET平台](https://learn.microsoft.com/en-us/dotnet/csharp/)最流行的语言，是一个免费、跨平台、开源的开发环境。 C# 程序可以在许多不同的设备上运行，从物联网 (IoT) 设备到云以及介于两者之间的任何地方。您可以为手机、台式机、笔记本电脑和服务器编写应用程序。



C# 是一种跨平台通用语言，可提高开发人员在编写高性能代码时的工作效率。 C# 是最流行的 .NET 语言，拥有数百万开发人员。 C# 在生态系统和所有 .NET workloads中拥有广泛的支持。基于面向对象的原则，它结合了其他范例的许多功能，尤其是函数式编程。底层功能支持高效场景，无需编写不安全代码。大多数 .NET 运行时和库都是用 C# 编写的，C# 的进步通常会使所有 .NET 开发人员受益。

## 你好世界

“Hello, World”程序传统上用于介绍编程语言。这是 C# 中的：

```csharp
// This line prints "Hello, World" 
Console.WriteLine("Hello, World");
```

以`//`开头的行是_单行注释_。 C# 单行注释以`//`开始，一直到当前行的末尾。 C#还支持_多行注释_。多行注释以`/*`开头，以`*/`结尾。 `Console`类（位于`System`命名空间中）的`WriteLine`方法生成程序的输出。此类由标准类库提供，默认情况下，每个 C# 程序都会自动引用该类。

前面的示例显示了“Hello, World”程序的一种形式，使用 top level statements。 C# 的早期版本要求您在方法中定义程序的入口点。此格式仍然有效，您将在许多现有的 C# 示例中看到它。您也应该熟悉此格式，如以下示例所示：

```csharp
using System;

class Hello
{
    static void Main()
    {
        // This line prints "Hello, World" 
        Console.WriteLine("Hello, World");
    }
}
```

现在您不必在控制台应用程序项目中显式包含`Main`方法。相反，您可以使用 top level statements 功能来最大限度地减少必须编写的代码。

top level statements 允许您直接在文件的根目录编写可执行代码，从而无需将代码包装在类或方法中。这意味着您可以创建程序而无需`Program`类和`Main`方法的仪式。在这种情况下，编译器会生成一个带有应用程序入口点方法`Program`类。生成的方法的名称不是`Main` ，它是代码无法直接引用的实现细节。

下面是一个_Program.cs_文件，它是 C# 10 中的完整 C# 程序：

```csharp
Console.WriteLine("Hello World!");
```

top level statements  允许你为小型实用程序（例如 Azure Functions 和 GitHub Actions）编写简单的程序。它们还使新 C# 程序员能够更轻松地开始学习和编写代码。

### 只有一个顶级文件 <a href="#only-one-top-level-file" id="only-one-top-level-file"></a>

一个应用程序必须只有一个入口点。一个项目只能有一个包含 top level statements  的文件。将top level statements  放入项目中的多个文件中会导致以下编译器错误：

CS8802 Only one compilation unit can have top-level statements.

### 程序解释

```csharp
using System;

class Hello
{
    static void Main()
    {
        // This line prints "Hello, World" 
        Console.WriteLine("Hello, World");
    }
}
```

此版本显示了您在程序中使用的构建块。 “Hello, World”程序以引用`System`命名空间的`using`指令开始。命名空间提供了组织 C# 程序和库的分层方式。命名空间包含类型和其他命名空间——例如， `System`命名空间包含许多类型，例如程序中引用的`Console`类，以及许多其他命名空间，例如`IO`和`Collections` 。引用给定命名空间的`using`指令允许无限制地使用属于该命名空间的成员类型。由于`using`指令，程序可以使用`Console.WriteLine`作为`System.Console.WriteLine`的简写。在前面的示例中，该命名空间被[隐式](https://learn.microsoft.com/en-us/dotnet/csharp/language-reference/keywords/using-directive#global-modifier)包含在内。

“Hello, World”程序声明的`Hello`类有一个成员，即名为`Main`方法。 `Main`方法是用`static`修饰符声明的。虽然实例方法可以使用关键字`this`引用特定的封闭对象实例，但静态方法的操作无需引用特定对象。按照约定，当没有顶级语句时，名为`Main`的静态方法将充当 C# 程序的[入口点](https://learn.microsoft.com/en-us/dotnet/csharp/fundamentals/program-structure/main-command-line)。
