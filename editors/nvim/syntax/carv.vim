" Vim syntax file for the Carv programming language
" Language: Carv
" Based on: editors/vscode/syntaxes/carv.tmLanguage.json

if exists("b:current_syntax")
  finish
endif

" Comments
syntax region carvCommentBlock start="/\*" end="\*/" contains=carvCommentBlock
syntax match carvCommentLine "//.*$"

" Strings
syntax region carvStringInterpolated start=+f"+ end=+"+ contains=carvStringEscape,carvInterpolation
syntax region carvString start=+"+ skip=+\\"+ end=+"+ contains=carvStringEscape
syntax region carvChar start=+'+ end=+'+ contains=carvStringEscape
syntax match carvStringEscape "\\\\."

" Interpolation in f-strings
syntax region carvInterpolation matchgroup=carvInterpolationDelimiter start="{" end="}" contained contains=carvString,carvNumber,carvKeyword,carvType,carvFunctionCall,carvOperator

" Numbers
syntax match carvFloat "\<\d\+\.\d\+\>"
syntax match carvInteger "\<\d\+\>"
syntax match carvHex "\<0x[0-9a-fA-F]\+\>"
syntax match carvBinary "\<0b[01]\+\>"

" Keywords - control flow
syntax keyword carvConditional if else match try
syntax keyword carvRepeat for in while loop
syntax keyword carvStatement return break continue

" Keywords - declarations
syntax keyword carvDeclaration fn let mut const class interface impl static pub priv type function

" Keywords - other
syntax keyword carvKeyword async await spawn chan select send recv new self super
syntax keyword carvKeyword require from import export module volatile packed unsafe asm
syntax keyword carvKeyword is as

" Boolean and nil literals
syntax keyword carvBoolean true false
syntax keyword carvConstant nil
syntax keyword carvConstant Ok Err

" Types - primitive
syntax keyword carvType void int float bool string char any Result ptr

" Types - sized
syntax keyword carvType u8 u16 u32 u64 i8 i16 i32 i64 f32 f64 usize isize

" Self keyword
syntax match carvSelf "\<self\>"

" Function definitions
syntax match carvFunctionDef "\<fn\s\+[a-zA-Z_]\w*" contains=carvDeclaration nextgroup=carvFunctionName
syntax match carvFunctionName "[a-zA-Z_]\w*" contained

" Function calls
syntax match carvFunctionCall "[a-zA-Z_]\w*\s*(" contains=carvFunctionCallName
syntax match carvFunctionCallName "[a-zA-Z_]\w*" contained

" Method calls
syntax match carvMethodCall "\.[a-zA-Z_]\w*\s*(" contains=carvMethodName
syntax match carvMethodName "\.[a-zA-Z_]\w*" contained

" Variable declarations
syntax match carvVarDecl "\<\(let\|mut\|const\)\s\+[a-zA-Z_]\w*" contains=carvDeclaration,carvVarName
syntax match carvVarName "[a-zA-Z_]\w*" contained

" Property access
syntax match carvProperty "\.[a-zA-Z_]\w*\>" contains=carvPropertyName
syntax match carvPropertyName "\.[a-zA-Z_]\w*" contained

" Type annotations
syntax match carvTypeAnnotation ":\s*[a-zA-Z_]\w*" contains=carvTypeColon,carvTypeName
syntax match carvTypeColon ":" contained
syntax match carvTypeName "[a-zA-Z_]\w*" contained

" Return type arrow
syntax match carvReturnArrow "->\s*[a-zA-Z_]\w*" contains=carvArrow,carvReturnType
syntax match carvArrow "->" contained
syntax match carvReturnType "[a-zA-Z_]\w*" contained

" Operators
syntax match carvOperator "[+\-*/%]"
syntax match carvOperator "==\|!=\|<=\|>=\|<\|>"
syntax match carvOperator "&&\|||\|!"
syntax match carvOperator "~\||\|\^"
syntax match carvOperator "+=\|-=\|*=\|/=\|%=\|&=\||=\|^="
syntax match carvOperator "="
syntax match carvArrow "->\|=>"
syntax match carvOperator "<-"
syntax match carvOperator "?" nextgroup=carvOperator
syntax match carvPipe "|>"

" New expression
syntax match carvNew "\<new\s\+[a-zA-Z_]\w*" contains=carvNewKeyword,carvNewType
syntax keyword carvNewKeyword new contained
syntax match carvNewType "[a-zA-Z_]\w*" contained

" Cast expression
syntax match carvCast "\<as\s\+[a-zA-Z_]\w*" contains=carvCastKeyword,carvCastType
syntax keyword carvCastKeyword as contained
syntax match carvCastType "[a-zA-Z_]\w*" contained

" Match arm arrow
syntax match carvMatchArm "=>"

" Semicolons
syntax match carvSemicolon ";"

" Highlights
highlight default link carvCommentBlock        Comment
highlight default link carvCommentLine         Comment
highlight default link carvStringInterpolated  String
highlight default link carvString              String
highlight default link carvChar                String
highlight default link carvStringEscape        SpecialChar
highlight default link carvInterpolation       Special
highlight default link carvInterpolationDelimiter Delimiter
highlight default link carvFloat               Float
highlight default link carvInteger             Number
highlight default link carvHex                 Number
highlight default link carvBinary              Number
highlight default link carvConditional         Conditional
highlight default link carvRepeat              Repeat
highlight default link carvStatement           Statement
highlight default link carvDeclaration         Keyword
highlight default link carvKeyword             Keyword
highlight default link carvBoolean             Boolean
highlight default link carvConstant            Constant
highlight default link carvType                Type
highlight default link carvSelf                Identifier
highlight default link carvFunctionName        Function
highlight default link carvFunctionCallName    Function
highlight default link carvFunctionDef         Function
highlight default link carvMethodName          Function
highlight default link carvMethodCall          Function
highlight default link carvVarName             Identifier
highlight default link carvVarDecl             Identifier
highlight default link carvPropertyName        Identifier
highlight default link carvProperty            Identifier
highlight default link carvTypeColon           Delimiter
highlight default link carvTypeAnnotation      Type
highlight default link carvTypeName            Type
highlight default link carvArrow               Operator
highlight default link carvReturnArrow         Type
highlight default link carvReturnType          Type
highlight default link carvOperator            Operator
highlight default link carvPipe                Operator
highlight default link carvNewKeyword          Keyword
highlight default link carvNewType             Type
highlight default link carvNew                 Type
highlight default link carvCastKeyword         Keyword
highlight default link carvCastType            Type
highlight default link carvCast                Type
highlight default link carvMatchArm            Operator
highlight default link carvSemicolon           Delimiter

let b:current_syntax = "carv"
