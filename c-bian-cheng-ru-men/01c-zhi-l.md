# c# 之旅

C#语言是[.NET平台](https://learn.microsoft.com/en-us/dotnet/csharp/)最流行的语言，是一个免费、跨平台、开源的开发环境。 C# 程序可以在不同的设备上运行，从物联网 (IoT) 设备到云以及介于两者之间的任何地方。您可以为手机、台式机、笔记本电脑和服务器编写应用程序。

## 你好世界

```csharp
// This line prints "Hello, World" 
Console.WriteLine("Hello, World");
```

以`//`开头的行是_单行注释_。 C# 单行注释以`//`开始，一直到当前行的末尾。 C#还支持_多行注释_。多行注释以`/*`开头，以`*/`结尾。 `Console`类（位于`System`命名空间中）的`WriteLine`方法生成程序的输出。此类由标准类库提供，默认情况下，每个 C# 程序都会自动引用该类。

前面的示例显示了“Hello, World”程序的一种形式，叫做 top level statements。 top level statements 允许您直接在文件的根目录编写可执行代码，从而无需将代码包装在类或方法中。这意味着您可以创建程序而无需`Program`类和`Main`方法的仪式。在这种情况下，编译器会生成一个带有应用程序入口点方法`Program`类。生成的方法的名称不是`Main` ，它是代码无法直接引用的实现细节。

C# 的早期版本要求您在方法中定义程序的入口点。此格式仍然有效，您将在许多现有的 C# 示例中看到它。您也应该熟悉此格式，如以下示例所示：

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

&#x20;“Hello, World”程序以引用`System`命名空间的`using`指令开始。命名空间提供了组织 C# 程序和库的分层方式。命名空间包含类型和其他命名空间——例如， `System`命名空间包含许多类型，例如程序中引用的`Console`类，以及许多其他命名空间，例如`IO`和`Collections` 。引用给定命名空间的`using`指令允许无限制地使用属于该命名空间的成员类型。

由于`using`指令，程序可以使用`Console.WriteLine`作为`System.Console.WriteLine`的简写。

“Hello, World”程序声明的`Hello`类有一个成员，即名为`Main`方法。 `Main`方法是用`static`修饰符声明的。虽然实例方法可以使用关键字`this`引用特定的封闭对象实例，但静态方法的操作无需引用特定对象。按照约定，当没有顶级语句时，名为`Main`的静态方法将充当 C# 程序的[入口点](https://learn.microsoft.com/en-us/dotnet/csharp/fundamentals/program-structure/main-command-line)。
